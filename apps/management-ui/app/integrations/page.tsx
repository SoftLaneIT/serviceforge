/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { useToast } from "@/components/ui/toast";
import {
  integrationsStore,
  parseOpenApiSpec,
  flattenSpec,
  sendRequest,
  HTTP_METHODS,
  type ParsedSpec,
  type FlatEndpoint,
  type RequestConfig,
  type RequestResult,
  type HttpMethod,
  type OpenApiParameter,
} from "@/lib/api/integrations";
import { formatDate } from "@/lib/utils";
import {
  Plug,
  Upload,
  Trash2,
  ChevronRight,
  ChevronDown,
  Play,
  Plus,
  X,
  Copy,
  Check,
  Globe,
  Code2,
  List,
  FileJson,
  Loader2,
} from "lucide-react";

// ---------------------------------------------------------------------------
// Method badge colours
// ---------------------------------------------------------------------------

const METHOD_COLORS: Record<string, string> = {
  get:     "bg-blue-50 text-blue-700 border-blue-200",
  post:    "bg-green-50 text-green-700 border-green-200",
  put:     "bg-amber-50 text-amber-700 border-amber-200",
  patch:   "bg-orange-50 text-orange-700 border-orange-200",
  delete:  "bg-red-50 text-red-700 border-red-200",
  head:    "bg-slate-50 text-slate-600 border-slate-200",
  options: "bg-purple-50 text-purple-700 border-purple-200",
};

function MethodBadge({ method }: { method: string }) {
  return (
    <span
      className={`inline-flex items-center rounded border px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider font-mono ${METHOD_COLORS[method] ?? "bg-slate-50 text-slate-600 border-slate-200"}`}
    >
      {method}
    </span>
  );
}

// ---------------------------------------------------------------------------
// Status badge for response
// ---------------------------------------------------------------------------

function StatusBadge({ status }: { status: number }) {
  const color =
    status < 200 ? "bg-slate-100 text-slate-600" :
    status < 300 ? "bg-green-100 text-green-700" :
    status < 400 ? "bg-blue-100 text-blue-700" :
    status < 500 ? "bg-amber-100 text-amber-700" :
                   "bg-red-100 text-red-700";
  return (
    <span className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-semibold ${color}`}>
      {status}
    </span>
  );
}

// ---------------------------------------------------------------------------
// KV editor row
// ---------------------------------------------------------------------------

interface KvRow { key: string; value: string; enabled: boolean }

function KvEditor({
  rows,
  onChange,
  placeholder,
}: {
  rows: KvRow[];
  onChange: (rows: KvRow[]) => void;
  placeholder?: string;
}) {
  const set = (i: number, patch: Partial<KvRow>) => {
    const next = rows.map((r, idx) => idx === i ? { ...r, ...patch } : r);
    onChange(next);
  };
  const remove = (i: number) => onChange(rows.filter((_, idx) => idx !== i));
  const add = () => onChange([...rows, { key: "", value: "", enabled: true }]);

  return (
    <div className="space-y-1.5">
      {rows.map((row, i) => (
        <div key={i} className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={row.enabled}
            onChange={(e) => set(i, { enabled: e.target.checked })}
            className="h-3.5 w-3.5 rounded border-slate-300 text-brand-600"
          />
          <input
            value={row.key}
            onChange={(e) => set(i, { key: e.target.value })}
            placeholder={placeholder ?? "Key"}
            className="flex-1 rounded border border-slate-200 px-2 py-1 text-xs font-mono bg-white focus:outline-none focus:ring-1 focus:ring-brand-400"
          />
          <input
            value={row.value}
            onChange={(e) => set(i, { value: e.target.value })}
            placeholder="Value"
            className="flex-1 rounded border border-slate-200 px-2 py-1 text-xs font-mono bg-white focus:outline-none focus:ring-1 focus:ring-brand-400"
          />
          <button onClick={() => remove(i)} className="text-slate-400 hover:text-red-500 transition-colors">
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
      <button
        onClick={add}
        className="flex items-center gap-1 text-xs text-slate-500 hover:text-brand-600 transition-colors"
      >
        <Plus className="h-3 w-3" /> Add row
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Request tester panel
// ---------------------------------------------------------------------------

function RequestTester({
  endpoint,
  baseUrl,
  onBaseUrlChange,
}: {
  endpoint: FlatEndpoint;
  baseUrl: string;
  onBaseUrlChange: (url: string) => void;
}) {
  const { operation, method, path } = endpoint;

  // Collect path params from spec + pattern
  const pathParamNames = Array.from(path.matchAll(/\{([^}]+)\}/g)).map((m) => m[1]);

  const specParams: OpenApiParameter[] = [
    ...(operation.parameters ?? []),
  ];

  const initQueryRows = specParams
    .filter((p) => p.in === "query")
    .map((p) => ({
      key: p.name,
      value: String(p.schema?.default ?? p.example ?? ""),
      enabled: true,
    }));

  const initHeaderRows = specParams
    .filter((p) => p.in === "header")
    .map((p) => ({ key: p.name, value: "", enabled: true }));

  const [pathParams, setPathParams] = useState<Record<string, string>>(
    Object.fromEntries(pathParamNames.map((n) => [n, ""])),
  );
  const [queryRows, setQueryRows] = useState<KvRow[]>(
    initQueryRows.length ? initQueryRows : [],
  );
  const [headerRows, setHeaderRows] = useState<KvRow[]>(
    initHeaderRows.length ? initHeaderRows : [{ key: "Content-Type", value: "application/json", enabled: true }],
  );
  const [body, setBody] = useState(() => {
    const rb = operation.requestBody;
    const ex = rb?.content?.["application/json"]?.example;
    if (ex) return JSON.stringify(ex, null, 2);
    const schema = rb?.content?.["application/json"]?.schema;
    if (schema) return JSON.stringify(schema, null, 2);
    return "";
  });
  const [contentType, setContentType] = useState("application/json");
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<RequestResult | null>(null);
  const [resultTab, setResultTab] = useState<"body" | "headers">("body");
  const [copied, setCopied] = useState(false);

  const send = async () => {
    setSending(true);
    setResult(null);
    try {
      const cfg: RequestConfig = {
        baseUrl,
        method,
        path,
        pathParams,
        queryParams: queryRows,
        headers: headerRows,
        body,
        contentType,
      };
      const r = await sendRequest(cfg);
      setResult(r);
    } catch (err) {
      setResult({
        status: 0,
        statusText: err instanceof Error ? err.message : "Network error",
        headers: {},
        body: "",
        durationMs: 0,
        size: 0,
      });
    } finally {
      setSending(false);
    }
  };

  const copyResult = () => {
    if (result) {
      navigator.clipboard.writeText(result.body);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    }
  };

  const prettyBody = (() => {
    if (!result?.body) return "";
    try { return JSON.stringify(JSON.parse(result.body), null, 2); } catch { return result.body; }
  })();

  const hasRequestBody = !["get", "head", "delete"].includes(method);

  return (
    <div className="space-y-4">
      {/* URL bar */}
      <div className="flex items-center gap-2">
        <MethodBadge method={method} />
        <input
          value={baseUrl}
          onChange={(e) => onBaseUrlChange(e.target.value)}
          placeholder="https://api.example.com"
          className="w-48 rounded-l border border-r-0 border-slate-200 bg-slate-50 px-2 py-1.5 text-xs font-mono text-slate-600 focus:outline-none"
        />
        <div className="flex-1 rounded-r border border-slate-200 bg-white px-2 py-1.5 text-xs font-mono text-slate-700 truncate">
          {path}
        </div>
        <Button
          size="sm"
          onClick={send}
          loading={sending}
          disabled={sending}
        >
          <Play className="h-3.5 w-3.5" />
          Send
        </Button>
      </div>

      {/* Path params */}
      {pathParamNames.length > 0 && (
        <div>
          <p className="text-xs font-medium text-slate-600 mb-2">Path Parameters</p>
          <div className="space-y-1.5">
            {pathParamNames.map((name) => (
              <div key={name} className="flex items-center gap-2">
                <span className="w-32 text-xs font-mono text-slate-500 shrink-0">{`{${name}}`}</span>
                <input
                  value={pathParams[name] ?? ""}
                  onChange={(e) =>
                    setPathParams((prev) => ({ ...prev, [name]: e.target.value }))
                  }
                  placeholder="value"
                  className="flex-1 rounded border border-slate-200 px-2 py-1 text-xs font-mono bg-white focus:outline-none focus:ring-1 focus:ring-brand-400"
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tabs: Query / Headers / Body */}
      <div>
        <div className="flex gap-4 border-b border-slate-100 mb-3">
          {(["Query", "Headers", ...(hasRequestBody ? ["Body"] : [])] as string[]).map((tab) => (
            <button
              key={tab}
              className="pb-2 text-xs font-medium text-slate-500 border-b-2 border-transparent data-[active=true]:border-brand-500 data-[active=true]:text-slate-900 transition-colors"
              data-active={tab === "Query" ? "true" : undefined}
              onClick={(e) => {
                const parent = e.currentTarget.parentElement!;
                parent.querySelectorAll("button").forEach((b) => b.removeAttribute("data-active"));
                e.currentTarget.setAttribute("data-active", "true");
              }}
            >
              {tab}
              {tab === "Query" && queryRows.filter((r) => r.enabled && r.key).length > 0 && (
                <span className="ml-1 text-[10px] rounded-full bg-brand-100 text-brand-700 px-1">
                  {queryRows.filter((r) => r.enabled && r.key).length}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* We use CSS show/hide via parent data-active attribute — simpler: always render, hide the non-active */}
        {/* Instead use local state for tab */}
      </div>

      {/* Query params */}
      <div>
        <p className="text-xs font-medium text-slate-600 mb-2">Query Parameters</p>
        <KvEditor rows={queryRows} onChange={setQueryRows} placeholder="param" />
      </div>

      {/* Headers */}
      <div>
        <p className="text-xs font-medium text-slate-600 mb-2">Headers</p>
        <KvEditor rows={headerRows} onChange={setHeaderRows} placeholder="Header-Name" />
      </div>

      {/* Body */}
      {hasRequestBody && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <p className="text-xs font-medium text-slate-600">Request Body</p>
            <select
              value={contentType}
              onChange={(e) => setContentType(e.target.value)}
              className="text-xs border border-slate-200 rounded px-1.5 py-0.5 text-slate-600 focus:outline-none"
            >
              <option value="application/json">application/json</option>
              <option value="application/x-www-form-urlencoded">form-urlencoded</option>
              <option value="multipart/form-data">multipart/form-data</option>
              <option value="text/plain">text/plain</option>
            </select>
          </div>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={6}
            placeholder='{ "key": "value" }'
            className="w-full rounded border border-slate-200 bg-white p-2 text-xs font-mono focus:outline-none focus:ring-1 focus:ring-brand-400 resize-y"
          />
        </div>
      )}

      {/* Response */}
      {result !== null && (
        <div className="rounded-lg border border-slate-200 overflow-hidden">
          <div className="flex items-center justify-between px-3 py-2 bg-slate-50 border-b border-slate-200">
            <div className="flex items-center gap-3">
              <StatusBadge status={result.status} />
              <span className="text-xs text-slate-500">{result.statusText}</span>
              {result.durationMs > 0 && (
                <span className="text-xs text-slate-400">{result.durationMs} ms</span>
              )}
              {result.size > 0 && (
                <span className="text-xs text-slate-400">{result.size} B</span>
              )}
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setResultTab("body")}
                className={`text-xs px-2 py-0.5 rounded ${resultTab === "body" ? "bg-white border border-slate-200 text-slate-800" : "text-slate-500"}`}
              >
                Body
              </button>
              <button
                onClick={() => setResultTab("headers")}
                className={`text-xs px-2 py-0.5 rounded ${resultTab === "headers" ? "bg-white border border-slate-200 text-slate-800" : "text-slate-500"}`}
              >
                Headers
              </button>
              <button onClick={copyResult} className="text-slate-400 hover:text-slate-700 transition-colors ml-1">
                {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
              </button>
            </div>
          </div>
          {resultTab === "body" ? (
            <pre className="text-xs font-mono p-3 bg-slate-900 text-green-400 overflow-x-auto max-h-64 scrollbar-thin">
              {prettyBody || <span className="text-slate-500 italic">empty body</span>}
            </pre>
          ) : (
            <div className="p-3 space-y-1 max-h-48 overflow-y-auto">
              {Object.entries(result.headers).map(([k, v]) => (
                <div key={k} className="flex gap-2 text-xs font-mono">
                  <span className="text-slate-500 shrink-0">{k}:</span>
                  <span className="text-slate-700 break-all">{v}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Endpoint detail panel
// ---------------------------------------------------------------------------

function EndpointPanel({
  endpoint,
  baseUrl,
  onBaseUrlChange,
}: {
  endpoint: FlatEndpoint;
  baseUrl: string;
  onBaseUrlChange: (url: string) => void;
}) {
  const { operation, method, path } = endpoint;
  const [tab, setTab] = useState<"tester" | "docs">("tester");

  return (
    <div className="flex-1 min-w-0">
      <div className="border-b border-slate-200 px-5 py-3 flex items-center gap-3">
        <MethodBadge method={method} />
        <code className="text-sm font-mono text-slate-800 flex-1 truncate">{path}</code>
        <div className="flex gap-1">
          <button
            onClick={() => setTab("tester")}
            className={`text-xs px-2.5 py-1 rounded ${tab === "tester" ? "bg-brand-50 text-brand-700 font-medium" : "text-slate-500 hover:bg-slate-50"}`}
          >
            <Play className="h-3 w-3 inline mr-1" />
            Try it
          </button>
          <button
            onClick={() => setTab("docs")}
            className={`text-xs px-2.5 py-1 rounded ${tab === "docs" ? "bg-brand-50 text-brand-700 font-medium" : "text-slate-500 hover:bg-slate-50"}`}
          >
            <FileJson className="h-3 w-3 inline mr-1" />
            Docs
          </button>
        </div>
      </div>

      <div className="p-5 overflow-y-auto max-h-[calc(100vh-200px)] space-y-4">
        {operation.summary && (
          <p className="text-sm font-medium text-slate-800">{operation.summary}</p>
        )}
        {operation.description && (
          <p className="text-xs text-slate-500 leading-relaxed">{operation.description}</p>
        )}

        {tab === "tester" ? (
          <RequestTester
            endpoint={endpoint}
            baseUrl={baseUrl}
            onBaseUrlChange={onBaseUrlChange}
          />
        ) : (
          <DocsView endpoint={endpoint} />
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Docs view
// ---------------------------------------------------------------------------

function DocsView({ endpoint }: { endpoint: FlatEndpoint }) {
  const { operation } = endpoint;

  const params = operation.parameters ?? [];
  const pathParams = params.filter((p) => p.in === "path");
  const queryParams = params.filter((p) => p.in === "query");
  const headerParams = params.filter((p) => p.in === "header");

  return (
    <div className="space-y-5 text-xs">
      {/* Parameters */}
      {params.length > 0 && (
        <div>
          <p className="font-semibold text-slate-700 mb-2">Parameters</p>
          <div className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
            {[...pathParams, ...queryParams, ...headerParams].map((p) => (
              <div key={`${p.in}-${p.name}`} className="flex gap-3 px-3 py-2">
                <span className="font-mono text-slate-800 shrink-0 w-32 truncate">{p.name}</span>
                <span className="text-slate-400 shrink-0 w-14">{p.in}</span>
                <span className="text-slate-400 shrink-0 w-16">{p.schema?.type ?? "—"}</span>
                {p.required && <Badge variant="error">required</Badge>}
                <span className="text-slate-500 flex-1">{p.description}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Request body */}
      {operation.requestBody && (
        <div>
          <p className="font-semibold text-slate-700 mb-2">Request Body</p>
          <div className="space-y-2">
            {Object.entries(operation.requestBody.content ?? {}).map(([ct, content]) => (
              <div key={ct}>
                <span className="font-mono text-slate-500">{ct}</span>
                {content.example !== undefined && (
                  <pre className="mt-1 p-2 bg-slate-50 rounded text-xs font-mono overflow-x-auto text-slate-700">
                    {JSON.stringify(content.example, null, 2)}
                  </pre>
                )}
                {content.example === undefined && content.schema && (
                  <pre className="mt-1 p-2 bg-slate-50 rounded text-xs font-mono overflow-x-auto text-slate-700">
                    {JSON.stringify(content.schema, null, 2)}
                  </pre>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Responses */}
      {operation.responses && (
        <div>
          <p className="font-semibold text-slate-700 mb-2">Responses</p>
          <div className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
            {Object.entries(operation.responses).map(([code, resp]) => (
              <div key={code} className="px-3 py-2 flex items-start gap-3">
                <StatusBadge status={Number(code)} />
                <span className="text-slate-600">{resp.description}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Import dialog
// ---------------------------------------------------------------------------

function ImportPanel({ onImport }: { onImport: (spec: ParsedSpec) => void }) {
  const { toast } = useToast();
  const fileRef = useRef<HTMLInputElement>(null);
  const [url, setUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState<"file" | "url" | "paste">("file");
  const [pasted, setPasted] = useState("");

  const importRaw = useCallback(async (raw: string, name: string) => {
    setLoading(true);
    try {
      const spec = await parseOpenApiSpec(raw, name);
      integrationsStore.save(spec);
      onImport(spec);
      toast(`Imported "${spec.name}" — ${Object.keys(spec.paths).length} paths`, "success");
    } catch (e) {
      toast(e instanceof Error ? e.message : "Parse error", "error");
    } finally {
      setLoading(false);
    }
  }, [onImport, toast]);

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const text = await file.text();
    importRaw(text, file.name);
  };

  const handleUrl = async () => {
    if (!url) return;
    setLoading(true);
    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      const text = await res.text();
      importRaw(text, url.split("/").pop() ?? "spec");
    } catch (e) {
      toast(e instanceof Error ? e.message : "Fetch error", "error");
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Import OpenAPI Specification</CardTitle>
        <CardDescription>
          Import OpenAPI 3.x or Swagger 2.x specs (JSON or YAML). Browse endpoints and test them directly.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Tab bar */}
        <div className="flex gap-1 border-b border-slate-100 pb-3">
          {(["file", "url", "paste"] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`px-3 py-1.5 text-xs rounded capitalize font-medium transition-colors ${tab === t ? "bg-brand-50 text-brand-700" : "text-slate-500 hover:bg-slate-50"}`}
            >
              {t === "file" ? "Upload File" : t === "url" ? "From URL" : "Paste JSON/YAML"}
            </button>
          ))}
        </div>

        {tab === "file" && (
          <div>
            <input
              ref={fileRef}
              type="file"
              accept=".json,.yaml,.yml"
              className="hidden"
              onChange={handleFile}
            />
            <button
              onClick={() => fileRef.current?.click()}
              className="w-full border-2 border-dashed border-slate-200 rounded-lg p-8 text-center hover:border-brand-300 hover:bg-brand-50/30 transition-colors group"
            >
              <Upload className="h-8 w-8 text-slate-300 group-hover:text-brand-400 mx-auto mb-2 transition-colors" />
              <p className="text-sm font-medium text-slate-600">Click to upload</p>
              <p className="text-xs text-slate-400 mt-1">openapi.json · openapi.yaml · swagger.json</p>
            </button>
          </div>
        )}

        {tab === "url" && (
          <div className="flex gap-2">
            <Input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://petstore.swagger.io/v2/swagger.json"
              className="flex-1"
            />
            <Button onClick={handleUrl} loading={loading} disabled={!url || loading}>
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Globe className="h-4 w-4" />}
              Fetch
            </Button>
          </div>
        )}

        {tab === "paste" && (
          <div className="space-y-2">
            <textarea
              value={pasted}
              onChange={(e) => setPasted(e.target.value)}
              rows={10}
              placeholder={'{\n  "openapi": "3.0.0",\n  "info": { "title": "My API", "version": "1.0" },\n  "paths": {}\n}'}
              className="w-full rounded border border-slate-200 p-2 text-xs font-mono focus:outline-none focus:ring-1 focus:ring-brand-400 resize-y"
            />
            <div className="flex justify-end">
              <Button
                onClick={() => importRaw(pasted, "pasted-spec")}
                loading={loading}
                disabled={!pasted.trim() || loading}
              >
                <Code2 className="h-4 w-4" />
                Parse &amp; Import
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// Spec list sidebar
// ---------------------------------------------------------------------------

function SpecList({
  specs,
  selectedId,
  onSelect,
  onDelete,
}: {
  specs: ParsedSpec[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  if (!specs.length) return null;

  return (
    <div className="space-y-1">
      {specs.map((spec) => (
        <div
          key={spec.id}
          className={`group flex items-center gap-2 rounded-lg px-3 py-2 cursor-pointer transition-colors ${selectedId === spec.id ? "bg-brand-50 text-brand-700" : "hover:bg-slate-50 text-slate-700"}`}
          onClick={() => onSelect(spec.id)}
        >
          <Plug className={`h-3.5 w-3.5 shrink-0 ${selectedId === spec.id ? "text-brand-500" : "text-slate-400"}`} />
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">{spec.name}</p>
            <p className="text-xs text-slate-400 truncate">v{spec.version} · {Object.keys(spec.paths).length} paths</p>
          </div>
          <button
            onClick={(e) => { e.stopPropagation(); onDelete(spec.id); }}
            className="opacity-0 group-hover:opacity-100 text-slate-400 hover:text-red-500 transition-all"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Endpoint browser
// ---------------------------------------------------------------------------

function EndpointBrowser({
  spec,
}: {
  spec: ParsedSpec;
}) {
  const [selectedEndpoint, setSelectedEndpoint] = useState<FlatEndpoint | null>(null);
  const [expandedTags, setExpandedTags] = useState<Set<string>>(new Set(["default"]));
  const [baseUrl, setBaseUrl] = useState(spec.servers[0]?.url ?? "");
  const [search, setSearch] = useState("");
  const [methodFilter, setMethodFilter] = useState<HttpMethod | "">("");

  const endpoints = flattenSpec(spec);

  const filtered = endpoints.filter((ep) => {
    const q = search.toLowerCase();
    const matchesSearch = !q ||
      ep.path.toLowerCase().includes(q) ||
      (ep.operation.summary ?? "").toLowerCase().includes(q) ||
      (ep.operation.operationId ?? "").toLowerCase().includes(q);
    const matchesMethod = !methodFilter || ep.method === methodFilter;
    return matchesSearch && matchesMethod;
  });

  // Group by first tag
  const grouped = new Map<string, FlatEndpoint[]>();
  for (const ep of filtered) {
    const tag = ep.tags[0] ?? "default";
    if (!grouped.has(tag)) grouped.set(tag, []);
    grouped.get(tag)!.push(ep);
  }

  const toggleTag = (tag: string) =>
    setExpandedTags((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) next.delete(tag);
      else next.add(tag);
      return next;
    });

  return (
    <div className="flex gap-0 border border-slate-200 rounded-xl overflow-hidden min-h-[500px]">
      {/* Endpoint list */}
      <div className="w-72 shrink-0 border-r border-slate-200 flex flex-col">
        {/* Search & filter */}
        <div className="p-3 border-b border-slate-100 space-y-2">
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search endpoints…"
            className="w-full rounded border border-slate-200 px-2.5 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-brand-400"
          />
          <div className="flex gap-1 flex-wrap">
            <button
              onClick={() => setMethodFilter("")}
              className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase border transition-colors ${!methodFilter ? "bg-slate-800 text-white border-slate-800" : "border-slate-200 text-slate-500 hover:bg-slate-50"}`}
            >
              ALL
            </button>
            {(["get", "post", "put", "patch", "delete"] as HttpMethod[]).map((m) => (
              <button
                key={m}
                onClick={() => setMethodFilter(methodFilter === m ? "" : m)}
                className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase border transition-colors ${methodFilter === m ? METHOD_COLORS[m] + " font-bold" : "border-slate-200 text-slate-500 hover:bg-slate-50"}`}
              >
                {m}
              </button>
            ))}
          </div>
        </div>

        {/* Grouped list */}
        <div className="flex-1 overflow-y-auto">
          {grouped.size === 0 && (
            <p className="text-xs text-slate-400 text-center py-8">No endpoints match your filter.</p>
          )}
          {Array.from(grouped.entries()).map(([tag, eps]) => (
            <div key={tag}>
              <button
                onClick={() => toggleTag(tag)}
                className="w-full flex items-center gap-2 px-3 py-2 text-xs font-semibold text-slate-500 uppercase tracking-wider hover:bg-slate-50 transition-colors"
              >
                {expandedTags.has(tag) ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                {tag}
                <span className="ml-auto text-[10px] text-slate-400">{eps.length}</span>
              </button>
              {expandedTags.has(tag) && eps.map((ep) => {
                const active = selectedEndpoint?.method === ep.method && selectedEndpoint?.path === ep.path;
                return (
                  <button
                    key={`${ep.method}:${ep.path}`}
                    onClick={() => setSelectedEndpoint(ep)}
                    className={`w-full flex items-center gap-2 px-3 py-2 text-left transition-colors ${active ? "bg-brand-50" : "hover:bg-slate-50"}`}
                  >
                    <MethodBadge method={ep.method} />
                    <span className={`text-xs font-mono truncate flex-1 ${active ? "text-brand-700" : "text-slate-700"}`}>
                      {ep.path}
                    </span>
                  </button>
                );
              })}
            </div>
          ))}
        </div>

        {/* Summary */}
        <div className="border-t border-slate-100 px-3 py-2 text-xs text-slate-400">
          {filtered.length} of {endpoints.length} endpoints
        </div>
      </div>

      {/* Detail pane */}
      {selectedEndpoint ? (
        <EndpointPanel
          endpoint={selectedEndpoint}
          baseUrl={baseUrl}
          onBaseUrlChange={setBaseUrl}
        />
      ) : (
        <div className="flex-1 flex items-center justify-center text-slate-400">
          <div className="text-center space-y-2">
            <List className="h-8 w-8 mx-auto opacity-30" />
            <p className="text-sm">Select an endpoint to test it</p>
          </div>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function IntegrationsPage() {
  const [specs, setSpecs] = useState<ParsedSpec[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [showImport, setShowImport] = useState(false);

  // Load from localStorage on mount
  useEffect(() => {
    setSpecs(integrationsStore.list());
  }, []);

  const selectedSpec = specs.find((s) => s.id === selectedId) ?? null;

  const handleImport = (spec: ParsedSpec) => {
    setSpecs(integrationsStore.list());
    setSelectedId(spec.id);
    setShowImport(false);
  };

  const handleDelete = (id: string) => {
    integrationsStore.delete(id);
    setSpecs(integrationsStore.list());
    if (selectedId === id) setSelectedId(null);
  };

  return (
    <Shell>
      <div className="space-y-5 max-w-7xl">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Integrations Hub</h2>
            <p className="text-sm text-slate-500 mt-0.5">
              Import OpenAPI specs, browse endpoints, and test your APIs interactively.
            </p>
          </div>
          <Button onClick={() => setShowImport((v) => !v)}>
            <Upload className="h-4 w-4" />
            Import Spec
          </Button>
        </div>

        {/* Import panel */}
        {showImport && (
          <ImportPanel onImport={handleImport} />
        )}

        {specs.length === 0 && !showImport ? (
          <EmptyState
            icon={Plug}
            title="No integrations yet"
            description="Import an OpenAPI or Swagger spec to browse and test your APIs."
            action={
              <Button onClick={() => setShowImport(true)}>
                <Upload className="h-4 w-4" />
                Import Spec
              </Button>
            }
          />
        ) : (
          <div className="flex gap-5">
            {/* Spec list */}
            {specs.length > 0 && (
              <aside className="w-56 shrink-0">
                <p className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2 px-1">
                  Saved Specs
                </p>
                <SpecList
                  specs={specs}
                  selectedId={selectedId}
                  onSelect={setSelectedId}
                  onDelete={handleDelete}
                />
              </aside>
            )}

            {/* Endpoint browser */}
            <div className="flex-1 min-w-0">
              {selectedSpec ? (
                <div className="space-y-3">
                  {/* Spec header */}
                  <div className="flex items-center gap-3">
                    <div>
                      <h3 className="text-sm font-semibold text-slate-900">{selectedSpec.name}</h3>
                      <p className="text-xs text-slate-400">
                        v{selectedSpec.version}
                        {selectedSpec.description && ` · ${selectedSpec.description}`}
                        {" · "}imported {formatDate(selectedSpec.importedAt)}
                      </p>
                    </div>
                    {selectedSpec.servers.length > 0 && (
                      <div className="ml-auto flex items-center gap-1.5">
                        <Globe className="h-3.5 w-3.5 text-slate-400" />
                        <span className="text-xs font-mono text-slate-500 truncate max-w-xs">
                          {selectedSpec.servers[0].url}
                        </span>
                        {selectedSpec.servers.length > 1 && (
                          <Badge variant="neutral">+{selectedSpec.servers.length - 1}</Badge>
                        )}
                      </div>
                    )}
                  </div>

                  <EndpointBrowser spec={selectedSpec} />
                </div>
              ) : (
                <EmptyState
                  icon={Plug}
                  title="Select a spec"
                  description="Choose an imported spec from the left to browse its endpoints."
                />
              )}
            </div>
          </div>
        )}
      </div>
    </Shell>
  );
}
