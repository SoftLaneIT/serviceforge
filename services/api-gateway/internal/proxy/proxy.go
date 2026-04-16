/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 * You may obtain a copy of the LICENSE at
 *
 * https://softlaneit.com/LICENSE.txt
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the LICENSE is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the LICENSE for the
 * specific language governing permissions and limitations
 * under the LICENSE.
 */

// Package proxy wraps httputil.ReverseProxy with gateway-specific behaviour:
// stripping hop-by-hop headers, forwarding the resolved X-Tenant-ID, and
// adding a structured error response on upstream failure.
package proxy

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// New returns an httputil.ReverseProxy that forwards requests to target.
// The Director:
//   - rewrites the request URL to the target host/scheme
//   - removes the Authorization header (the key was already validated)
//   - preserves all other headers including X-Tenant-ID injected by the auth
//     middleware
//
// The ErrorHandler returns a structured JSON 502 on upstream failure.
func New(target *url.URL, log *slog.Logger) *httputil.ReverseProxy {
	rp := httputil.NewSingleHostReverseProxy(target)

	// Wrap the default Director so we can post-process the request.
	defaultDirector := rp.Director
	rp.Director = func(req *http.Request) {
		defaultDirector(req)
		// Remove Authorization — the downstream services trust X-Tenant-ID,
		// not raw API keys.
		req.Header.Del("Authorization")
		// Ensure the Host header matches the target (some upstreams require it).
		req.Host = target.Host
	}

	// Strip CORS headers from the upstream response.  The gateway's CORS
	// middleware (outermost layer) is responsible for setting them on the
	// final client response.  If the upstream also sets them — which it does
	// when the backend services include their own CORS middleware for direct
	// health-check calls — httputil.ReverseProxy would copy them, producing
	// duplicate Access-Control-Allow-Origin values that browsers reject.
	rp.ModifyResponse = func(res *http.Response) error {
		res.Header.Del("Access-Control-Allow-Origin")
		res.Header.Del("Access-Control-Allow-Methods")
		res.Header.Del("Access-Control-Allow-Headers")
		res.Header.Del("Access-Control-Allow-Credentials")
		res.Header.Del("Access-Control-Max-Age")
		res.Header.Del("Access-Control-Expose-Headers")
		return nil
	}

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("upstream error",
			slog.String("target", target.String()),
			slog.String("path", r.URL.Path),
			slog.Any("error", err),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "upstream service unavailable"})
	}

	return rp
}
