/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { use, useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { PageSpinner } from "@/components/ui/spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { useToast } from "@/components/ui/toast";
import { tenantsApi } from "@/lib/api/tenants";
import { configApi } from "@/lib/api/config";
import { SchemaForm } from "@/components/config/schema-form";
import { formatDate } from "@/lib/utils";
import { Settings2, Save, History, Code2, RefreshCw, ArrowLeft } from "lucide-react";

export default function TenantConfigPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { toast } = useToast();
  const qc = useQueryClient();
  const [selectedModule, setSelectedModule] = useState("");
  const [viewMode, setViewMode] = useState<"form" | "json">("form");
  const [configText, setConfigText] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [showSchema, setShowSchema] = useState(false);

  const tenantQuery = useQuery({
    queryKey: ["tenant", id],
    queryFn: () => tenantsApi.get(id),
  });

  const modulesQuery = useQuery({
    queryKey: ["modules"],
    queryFn: () => configApi.listModules(),
  });
  const modules = modulesQuery.data?.data ?? [];

  const configQuery = useQuery({
    queryKey: ["config", id, selectedModule],
    queryFn: () => configApi.getConfig(selectedModule, id),
    enabled: !!selectedModule,
  });

  useEffect(() => {
    if (configQuery.data) {
      setConfigText(JSON.stringify(configQuery.data.config, null, 2));
    }
  }, [configQuery.data]);

  const activeModule = modules.find((m) => m.module === selectedModule);

  const historyQuery = useQuery({
    queryKey: ["config-history", id, selectedModule],
    queryFn: () => configApi.getHistory(selectedModule, id),
    enabled: !!(selectedModule && showHistory),
  });

  const upsertMutation = useMutation({
    mutationFn: (config: Record<string, unknown>) =>
      configApi.upsertConfig(selectedModule, config, id),
    onSuccess: () => {
      toast("Configuration saved", "success");
      qc.invalidateQueries({ queryKey: ["config", id, selectedModule] });
      qc.invalidateQueries({ queryKey: ["config-history", id, selectedModule] });
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const handleSaveJson = () => {
    try {
      const parsed = JSON.parse(configText);
      upsertMutation.mutate(parsed);
    } catch {
      toast("Invalid JSON", "error");
    }
  };

  const isValidJson = (() => {
    try { JSON.parse(configText); return true; } catch { return false; }
  })();

  return (
    <Shell>
      <div className="space-y-5 max-w-3xl">
        <div className="flex items-center gap-3">
          <Link href={`/tenants/${id}`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4" />
              {tenantQuery.data?.name ?? "Tenant"}
            </Button>
          </Link>
        </div>

        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Configuration</h2>
            <p className="text-sm text-slate-500 mt-0.5">
              Per-module config for{" "}
              <span className="font-medium">{tenantQuery.data?.name ?? id}</span>
            </p>
          </div>

          {/* Module pills */}
          {modules.length > 0 && (
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
          )}
        </div>

        {!selectedModule ? (
          <EmptyState
            icon={Settings2}
            title="Select a module"
            description="Choose a module above to view and edit its configuration."
          />
        ) : configQuery.isLoading ? (
          <PageSpinner />
        ) : (
          <div className="space-y-5">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="capitalize">{selectedModule}</CardTitle>
                    <CardDescription>
                      {configQuery.data?.isDefault
                        ? "Using schema defaults — no custom config saved yet"
                        : `Updated ${formatDate(configQuery.data?.updatedAt)}`}
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    {configQuery.data?.isDefault && (
                      <Badge variant="neutral">default</Badge>
                    )}
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
                      onClick={() => qc.invalidateQueries({ queryKey: ["config", id, selectedModule] })}
                    >
                      <RefreshCw className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                {viewMode === "form" && activeModule?.schema?.properties ? (
                  <SchemaForm
                    schema={activeModule.schema}
                    defaults={activeModule.defaults}
                    savedConfig={configQuery.data?.config ?? {}}
                    onSave={(config) => upsertMutation.mutate(config)}
                    saving={upsertMutation.isPending}
                  />
                ) : (
                  <>
                    <textarea
                      value={configText}
                      onChange={(e) => setConfigText(e.target.value)}
                      rows={12}
                      className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-brand-500"
                    />
                    {!isValidJson && configText.length > 0 && (
                      <p className="text-xs text-red-600">Invalid JSON</p>
                    )}
                    <div className="flex items-center justify-end">
                      <Button
                        disabled={!isValidJson || !configText || upsertMutation.isPending}
                        loading={upsertMutation.isPending}
                        onClick={handleSaveJson}
                      >
                        <Save className="h-4 w-4" />
                        Save
                      </Button>
                    </div>
                  </>
                )}

                <div className="flex gap-2 pt-2 border-t border-slate-100">
                  <Button variant="ghost" size="sm" onClick={() => setShowSchema((v) => !v)}>
                    <Code2 className="h-3.5 w-3.5" />
                    {showSchema ? "Hide Schema" : "Schema"}
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setShowHistory((v) => !v)}>
                    <History className="h-3.5 w-3.5" />
                    {showHistory ? "Hide History" : "History"}
                  </Button>
                </div>
              </CardContent>
            </Card>

            {showSchema && activeModule && (
              <Card>
                <CardHeader>
                  <CardTitle>JSON Schema — {selectedModule}</CardTitle>
                </CardHeader>
                <CardContent>
                  <pre className="text-xs font-mono bg-slate-900 text-green-400 rounded-lg p-4 overflow-x-auto scrollbar-thin max-h-64">
                    {JSON.stringify(activeModule.schema, null, 2)}
                  </pre>
                </CardContent>
              </Card>
            )}

            {showHistory && (
              <Card>
                <CardHeader>
                  <CardTitle>Change History</CardTitle>
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
                            <span className="text-xs text-slate-500">
                              {formatDate(entry.changedAt)}
                            </span>
                            <span className="text-xs font-mono text-slate-400">
                              {entry.changedBy}
                            </span>
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
            )}
          </div>
        )}
      </div>
    </Shell>
  );
}
