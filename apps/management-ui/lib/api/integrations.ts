/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

// 
// OpenAPI types (subset we actually need for the endpoint browser)
// 

export interface OpenApiInfo {
  title: string;
  version: string;
  description?: string;
}

export interface OpenApiServer {
  url: string;
  description?: string;
}

export interface OpenApiParameter {
  name: string;
  in: "query" | "header" | "path" | "cookie";
  required?: boolean;
  description?: string;
  schema?: { type?: string; example?: unknown; default?: unknown };
  example?: unknown;
}

export interface OpenApiRequestBody {
  description?: string;
  required?: boolean;
  content?: Record<string, { schema?: Record<string, unknown>; example?: unknown }>;
}

export interface OpenApiResponse {
  description?: string;
  content?: Record<string, { schema?: Record<string, unknown>; example?: unknown }>;
}

export interface OpenApiOperation {
  operationId?: string;
  summary?: string;
  description?: string;
  tags?: string[];
  parameters?: OpenApiParameter[];
  requestBody?: OpenApiRequestBody;
  responses?: Record<string, OpenApiResponse>;
  security?: Record<string, string[]>[];
}

export interface OpenApiPathItem {
  get?: OpenApiOperation;
  post?: OpenApiOperation;
  put?: OpenApiOperation;
  patch?: OpenApiOperation;
  delete?: OpenApiOperation;
  head?: OpenApiOperation;
  options?: OpenApiOperation;
  parameters?: OpenApiParameter[];
}

export interface ParsedSpec {
  id: string;           // local UUID
  name: string;         // extracted from info.title
  version: string;
  description: string;
  servers: OpenApiServer[];
  paths: Record<string, OpenApiPathItem>;
  rawJson: string;      // serialised original
  importedAt: string;   // ISO timestamp
}

// 
// Derived: flat endpoint list for browsing
// 

export const HTTP_METHODS = ["get", "post", "put", "patch", "delete", "head", "options"] as const;
export type HttpMethod = typeof HTTP_METHODS[number];

export interface FlatEndpoint {
  method: HttpMethod;
  path: string;
  operation: OpenApiOperation;
  tags: string[];
}

export function flattenSpec(spec: ParsedSpec): FlatEndpoint[] {
  const endpoints: FlatEndpoint[] = [];
  for (const [path, item] of Object.entries(spec.paths ?? {})) {
    for (const method of HTTP_METHODS) {
      const op = item[method];
      if (op) {
        endpoints.push({
          method,
          path,
          operation: op,
          tags: op.tags ?? ["default"],
        });
      }
    }
  }
  return endpoints;
}

// 
// Saved integrations (localStorage — no backend needed for v1)
// 

const STORAGE_KEY = "sf_integrations";

function loadAll(): ParsedSpec[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as ParsedSpec[]) : [];
  } catch {
    return [];
  }
}

function saveAll(specs: ParsedSpec[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(specs));
}

export const integrationsStore = {
  list(): ParsedSpec[] {
    return loadAll();
  },

  get(id: string): ParsedSpec | undefined {
    return loadAll().find((s) => s.id === id);
  },

  save(spec: ParsedSpec): void {
    const all = loadAll();
    const idx = all.findIndex((s) => s.id === spec.id);
    if (idx >= 0) all[idx] = spec;
    else all.unshift(spec);
    saveAll(all);
  },

  delete(id: string): void {
    saveAll(loadAll().filter((s) => s.id !== id));
  },
};

// 
// Parser: JSON or YAML → ParsedSpec
// 

export async function parseOpenApiSpec(
  raw: string,
  filename: string,
): Promise<ParsedSpec> {
  let parsed: Record<string, unknown>;

  const trimmed = raw.trim();
  if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
    parsed = JSON.parse(trimmed);
  } else {
    // YAML
    const yaml = await import("js-yaml");
    parsed = yaml.load(trimmed) as Record<string, unknown>;
  }

  // Support both OpenAPI 3.x and Swagger 2.x
  const info = (parsed.info ?? {}) as Record<string, string>;
  const title = info.title ?? filename.replace(/\.[^.]+$/, "");
  const version = info.version ?? "unknown";
  const description = (info.description as string | undefined) ?? "";

  // Normalise servers
  let servers: OpenApiServer[] = [];
  if (Array.isArray(parsed.servers)) {
    servers = parsed.servers as OpenApiServer[];
  } else if (parsed.host) {
    // Swagger 2.x
    const scheme = Array.isArray(parsed.schemes) ? parsed.schemes[0] : "https";
    servers = [{ url: `${scheme}://${parsed.host}${parsed.basePath ?? ""}` }];
  }

  const paths = (parsed.paths ?? {}) as Record<string, OpenApiPathItem>;

  return {
    id: crypto.randomUUID(),
    name: title,
    version,
    description,
    servers,
    paths,
    rawJson: JSON.stringify(parsed, null, 2),
    importedAt: new Date().toISOString(),
  };
}

// 
// Request tester
// 

export interface RequestConfig {
  baseUrl: string;
  method: HttpMethod;
  path: string;
  pathParams: Record<string, string>;
  queryParams: Array<{ key: string; value: string; enabled: boolean }>;
  headers: Array<{ key: string; value: string; enabled: boolean }>;
  body: string;
  contentType: string;
}

export interface RequestResult {
  status: number;
  statusText: string;
  headers: Record<string, string>;
  body: string;
  durationMs: number;
  size: number;
}

export async function sendRequest(cfg: RequestConfig): Promise<RequestResult> {
  // Resolve path params
  let resolvedPath = cfg.path;
  for (const [key, val] of Object.entries(cfg.pathParams)) {
    resolvedPath = resolvedPath.replace(`{${key}}`, encodeURIComponent(val));
  }

  // Build query string
  const qs = new URLSearchParams();
  for (const p of cfg.queryParams) {
    if (p.enabled && p.key) qs.append(p.key, p.value);
  }
  const query = qs.toString() ? `?${qs}` : "";

  const url = `${cfg.baseUrl.replace(/\/$/, "")}${resolvedPath}${query}`;

  // Build headers
  const headers: Record<string, string> = {};
  for (const h of cfg.headers) {
    if (h.enabled && h.key) headers[h.key] = h.value;
  }
  if (cfg.body && cfg.contentType) {
    headers["Content-Type"] = cfg.contentType;
  }

  const hasBody = cfg.body &&
    !["GET", "HEAD", "DELETE"].includes(cfg.method.toUpperCase());

  const start = performance.now();
  const res = await fetch(url, {
    method: cfg.method.toUpperCase(),
    headers,
    body: hasBody ? cfg.body : undefined,
  });
  const durationMs = Math.round(performance.now() - start);

  const responseText = await res.text();
  const responseHeaders: Record<string, string> = {};
  res.headers.forEach((val, key) => { responseHeaders[key] = val; });

  return {
    status: res.status,
    statusText: res.statusText,
    headers: responseHeaders,
    body: responseText,
    durationMs,
    size: new TextEncoder().encode(responseText).length,
  };
}
