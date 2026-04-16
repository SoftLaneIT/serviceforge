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

// Package validator validates a configuration map against a JSON Schema
// (draft-07) loaded from the database.
package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)
// bytes is used for schema AddResource below

// Validate checks that cfg conforms to schemaDoc (a JSON Schema draft-07
// object).  It returns a human-readable description of all violations on
// failure, or nil when the config is valid.
func Validate(schemaDoc map[string]any, cfg map[string]any) error {
	// Serialise the schema to JSON so we can feed it to the jsonschema compiler.
	schemaJSON, err := json.Marshal(schemaDoc)
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	// Serialise the config so jsonschema can decode it through its own pipeline.
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	const schemaURL = "schema.json"
	if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaJSON)); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	// Decode cfg back to interface{} so the library receives a typed value,
	// not an io.Reader (v5.3.1 does not accept io.Reader in Validate).
	var cfgDecoded interface{}
	if err := json.Unmarshal(cfgJSON, &cfgDecoded); err != nil {
		return fmt.Errorf("re-decode config: %w", err)
	}
	if err := schema.Validate(cfgDecoded); err != nil {
		// Flatten the validation error tree into a readable string.
		return fmt.Errorf("config validation failed: %s", flattenErrors(err))
	}
	return nil
}

// flattenErrors converts a jsonschema.ValidationError tree into a short
// comma-separated list of the leaf messages.
func flattenErrors(err error) string {
	var ve *jsonschema.ValidationError
	if !isValidationError(err, &ve) {
		return err.Error()
	}

	var msgs []string
	collectLeaves(ve, &msgs)
	if len(msgs) == 0 {
		return err.Error()
	}
	return strings.Join(msgs, "; ")
}

func isValidationError(err error, out **jsonschema.ValidationError) bool {
	ve, ok := err.(*jsonschema.ValidationError)
	if ok {
		*out = ve
	}
	return ok
}

func collectLeaves(ve *jsonschema.ValidationError, msgs *[]string) {
	if len(ve.Causes) == 0 {
		loc := ve.InstanceLocation
		if loc == "" {
			loc = "(root)"
		}
		*msgs = append(*msgs, fmt.Sprintf("%s: %s", loc, ve.Message))
		return
	}
	for _, cause := range ve.Causes {
		collectLeaves(cause, msgs)
	}
}
