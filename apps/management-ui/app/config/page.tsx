/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageSpinner } from "@/components/ui/spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { useToast } from "@/components/ui/toast";
import { useTenant } from "@/lib/context/tenant-context";
import { configApi } from "@/lib/api/config";
import { SchemaForm } from "@/components/config/schema-form";
import { formatDate } from "@/lib/utils";
import {
  Settings2,
  History,
  Code2,
  RefreshCw,
  ChevronDown,
  Layers,
} from "lucide-react";

// 
// History panel
// 

function HistoryPanel({ module, tenantId }: { module: string; tenantId: string }) {
  const historyQuery = useQuery({
    queryKey: ["config-history", tenantId, module],
    queryFn: () => configApi.getHistory(module, tenantId),
  });

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <History className="h-4 w-4 text-slate-400" />
          <CardTitle>Change History</CardTitle>
        </div>
        <CardDescription>Previous config versions for {module}</CardDescription>
      </CardHeader>
      <CardContent>
        {historyQuery.isLoading ? (
          <PageSpinner />
        ) : !historyQuery.data?.data?.length ? (
          <p className="text-sm text-slate-500">No history yet.</p>
        ) : (
          <div className="space-y-3">
            {historyQuery.data.data.map((entry) => (
              <div key={entry.id} className="rounded-lg border border-slate-100 p-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-slate-500">{formatDate(entry.changedAt)}</span>
                  <span className="text-xs font-mono text-slate-400">{entry.changedBy}</span>
                </div>
                <pre className="text-xs font-mono bg-slate-50 rounded p-2 overflow-x-auto scrollbar-thin">
                  {JSON.stringify(entry.config, null, 2)}
                </pre>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// 
// Main page
// 

export default function ConfigPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const { activeTenant } = useTenant();

  const [selectedTenantId, setSelectedTenantId] = useState(activeTenant?.id ?? "");
  const [selectedModule, setSelectedModule] = useState("");
  const [viewMode, setViewMode] = useState<"form" | "json">("form");
  const [jsonText, setJsonText] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [showSchema, setShowSchema] = useState(false);

  // Sync tenant switcher → local selection
  useEffect(() => {
    if (activeTenant?.id) setSelectedTenantId(activeTenant.id);
  }, [activeTenant?.id]);

  // Fetch modules list
  const modulesQuery = useQuery({
    queryKey: ["modules"],
    queryFn: () => configApi.listModules(),
  });
  const modules = modulesQuery.data?.data ?? [];

  // Fetch current config for selected tenant+module
  const configQuery = useQuery({
    queryKey: ["config", selectedTenantId, selectedModule],
    queryFn: () => configApi.getConfig(selectedModule, selectedTenantId),
    enabled: !!(selectedTenantId && selectedModule),
  });

  // Sync JSON editor when config loads
  useEffect(() => {
    if (configQuery.data) {
      setJsonText(JSON.stringify(configQuery.data.config, null, 2));
    }
  }, [configQuery.data]);

  // Active module full object (has schema + defaults)
  const activeModule = modules.find((m) => m.module === selectedModule);

  const upsertMutation = useMutation({
    mutationFn: (config: Record<string, unknown>) =>
      configApi.upsertConfig(selectedModule, config, selectedTenantId),
    onSuccess: () => {
      toast("Configuration saved", "success");
      qc.invalidateQueries({ queryKey: ["config", selectedTenantId, selectedModule] });
      qc.invalidateQueries({ queryKey: ["config-history", selectedTenantId, selectedModule] });
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const handleSaveJson = () => {
    try {
      const parsed = JSON.parse(jsonText);
      upsertMutation.mutate(parsed);
    } catch {
      toast("Invalid JSON", "error");
    }
  };

  const isValidJson = (() => {
    try { JSON.parse(jsonText); return true; } catch { return false; }
  })();

  const noSelection = !selectedTenantId || !selectedModule;

  return (
    <Shell>
      <div className="space-y-5 max-w-4xl">
        <div>
          <h2 className="text-lg font-semibold text-slate-900">Configuration</h2>
          <p className="text-sm text-slate-500 mt-0.5">
            Per-tenant module settings — select a tenant from the header or below.
          </p>
        </div>

        {/* Selectors row */}
        <div className="flex gap-3 flex-wrap items-end">
          {/* Tenant select — only shown if no active tenant from switcher */}
          <div className="flex flex-col gap-1">
            <label className="text-xs font-medium text-slate-500 uppercase tracking-wider">Tenant</label>
            <TenantSelect
              selectedId={selectedTenantId}
              onChange={(id) => { setSelectedTenantId(id); setShowHistory(false); }}
            />
          </div>

          {/* Module pills */}
          {modules.length > 0 && (
            <div className="flex flex-col gap-1">
              <label className="text-xs font-medium text-slate-500 uppercase tracking-wider">Module</label>
              <div className="flex gap-2 flex-wrap">
                {modules.map((m) => (
                  <button
                    key={m.module}
                    onClick={() => {
                      setSelectedModule(m.module);
                      setShowHistory(false);
                      setShowSchema(false);
                    }}
                    className={`rounded-full px-3 py-1 text-sm font-medium border transition-colors ${
                      selectedModule === m.module
                        ? "bg-brand-600 text-white border-brand-600"
                        : "bg-white text-slate-600 border-slate-200 hover:border-brand-300 hover:text-brand-600"
                    }`}
                  >
                    {m.module}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        {noSelection ? (
          <EmptyState
            icon={Settings2}
            title="Select a tenant and module"
            description="Choose a tenant from the header switcher (or the dropdown above) and pick a module to configure."
          />
        ) : configQuery.isLoading ? (
          <PageSpinner />
        ) : (
          <div className="space-y-5">
            {/* Config editor card */}
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="capitalize">{selectedModule}</CardTitle>
                    <CardDescription>
                      {configQuery.data?.isDefault
                        ? "Using schema defaults — no custom config saved yet"
                        : `Last updated ${formatDate(configQuery.data?.updatedAt)}`}
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    {configQuery.data?.isDefault && <Badge variant="neutral">default</Badge>}
                    {/* View mode toggle */}
                    <div className="flex rounded-lg border border-slate-200 overflow-hidden">
                      <button
                        onClick={() => setViewMode("form")}
                        className={`px-2.5 py-1 text-xs font-medium transition-colors ${viewMode === "form" ? "bg-brand-50 text-brand-700" : "bg-white text-slate-500 hover:bg-slate-50"}`}
                      >
                        Form
                      </button>
                      <button
                        onClick={() => setViewMode("json")}
                        className={`px-2.5 py-1 text-xs font-medium border-l border-slate-200 transition-colors ${viewMode === "json" ? "bg-brand-50 text-brand-700" : "bg-white text-slate-500 hover:bg-slate-50"}`}
                      >
                        JSON
                      </button>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => qc.invalidateQueries({ queryKey: ["config", selectedTenantId, selectedModule] })}
                    >
                      <RefreshCw className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {viewMode === "form" && activeModule?.schema?.properties ? (
                  <SchemaForm
                    schema={activeModule.schema}
                    defaults={activeModule.defaults}
                    savedConfig={configQuery.data?.config ?? {}}
                    onSave={(config) => upsertMutation.mutate(config)}
                    saving={upsertMutation.isPending}
                  />
                ) : (
                  <div className="space-y-3">
                    <textarea
                      value={jsonText}
                      onChange={(e) => setJsonText(e.target.value)}
                      rows={14}
                      className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-brand-500"
                    />
                    {!isValidJson && jsonText.length > 0 && (
                      <p className="text-xs text-red-600">Invalid JSON</p>
                    )}
                    <div className="flex items-center justify-between">
                      <div className="flex gap-2">
                        <Button variant="ghost" size="sm" onClick={() => setShowSchema((v) => !v)}>
                          <Code2 className="h-3.5 w-3.5" />
                          {showSchema ? "Hide Schema" : "View Schema"}
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => setShowHistory((v) => !v)}>
                          <History className="h-3.5 w-3.5" />
                          {showHistory ? "Hide History" : "View History"}
                        </Button>
                      </div>
                      <Button
                        disabled={!isValidJson || !jsonText || upsertMutation.isPending}
                        loading={upsertMutation.isPending}
                        onClick={handleSaveJson}
                      >
                        Save Configuration
                      </Button>
                    </div>
                  </div>
                )}

                {/* Schema / History toggles available in form mode too */}
                {viewMode === "form" && (
                  <div className="flex gap-2 mt-4 pt-3 border-t border-slate-100">
                    <Button variant="ghost" size="sm" onClick={() => setShowSchema((v) => !v)}>
                      <Code2 className="h-3.5 w-3.5" />
                      {showSchema ? "Hide Schema" : "View Schema"}
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => setShowHistory((v) => !v)}>
                      <History className="h-3.5 w-3.5" />
                      {showHistory ? "Hide History" : "View History"}
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Schema viewer */}
            {showSchema && activeModule && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <Code2 className="h-4 w-4 text-slate-400" />
                    <CardTitle>JSON Schema — {selectedModule}</CardTitle>
                  </div>
                </CardHeader>
                <CardContent>
                  <pre className="text-xs font-mono bg-slate-900 text-green-400 rounded-lg p-4 overflow-x-auto scrollbar-thin max-h-72">
                    {JSON.stringify(activeModule.schema, null, 2)}
                  </pre>
                </CardContent>
              </Card>
            )}

            {showHistory && (
              <HistoryPanel module={selectedModule} tenantId={selectedTenantId} />
            )}
          </div>
        )}
      </div>
    </Shell>
  );
}

// 
// Inline tenant selector (used when no active tenant from header switcher)
// 

function TenantSelect({
  selectedId,
  onChange,
}: {
  selectedId: string;
  onChange: (id: string) => void;
}) {
  const { activeTenant } = useTenant();
  const tenantsQuery = useQuery({
    queryKey: ["tenants", "active-all"],
    queryFn: () => import("@/lib/api/tenants").then((m) => m.tenantsApi.list({ status: "active", limit: 200 })),
    staleTime: 60_000,
  });
  const tenants = tenantsQuery.data?.data ?? [];

  // If active tenant is set from header, show a locked display
  if (activeTenant) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-brand-200 bg-brand-50 px-3 py-1.5">
        <div className="flex h-5 w-5 items-center justify-center rounded bg-brand-100">
          <span className="text-[10px] font-bold text-brand-700">
            {activeTenant.name.slice(0, 2).toUpperCase()}
          </span>
        </div>
        <span className="text-sm font-medium text-brand-800">{activeTenant.name}</span>
        <span className="text-xs font-mono text-brand-500">({activeTenant.slug})</span>
      </div>
    );
  }

  return (
    <select
      value={selectedId}
      onChange={(e) => onChange(e.target.value)}
      className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-brand-500 min-w-[200px]"
    >
      <option value="">Select a tenant…</option>
      {tenants.map((t) => (
        <option key={t.id} value={t.id}>
          {t.name} ({t.slug})
        </option>
      ))}
    </select>
  );
}
