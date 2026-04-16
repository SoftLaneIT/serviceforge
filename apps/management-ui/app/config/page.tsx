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
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { PageSpinner } from "@/components/ui/spinner";
import { EmptyState } from "@/components/ui/empty-state";
import { useToast } from "@/components/ui/toast";
import { tenantsApi } from "@/lib/api/tenants";
import { configApi } from "@/lib/api/config";
import { formatDate } from "@/lib/utils";
import { Settings2, Save, History, Code2, RefreshCw } from "lucide-react";

function HistoryPanel({
  module,
  tenantId,
}: {
  module: string;
  tenantId: string;
}) {
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

export default function ConfigPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [selectedTenantId, setSelectedTenantId] = useState("");
  const [selectedModule, setSelectedModule] = useState("");
  const [configText, setConfigText] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [showSchema, setShowSchema] = useState(false);

  const tenantsQuery = useQuery({
    queryKey: ["tenants", "active-all"],
    queryFn: () => tenantsApi.list({ status: "active", limit: 200 }),
  });

  const modulesQuery = useQuery({
    queryKey: ["modules"],
    queryFn: () => configApi.listModules(),
  });

  const configQuery = useQuery({
    queryKey: ["config", selectedTenantId, selectedModule],
    queryFn: () => configApi.getConfig(selectedModule, selectedTenantId),
    enabled: !!(selectedTenantId && selectedModule),
  });

  useEffect(() => {
    if (configQuery.data) {
      setConfigText(JSON.stringify(configQuery.data.config, null, 2));
    }
  }, [configQuery.data]);

  const schemaQuery = useQuery({
    queryKey: ["schema", selectedModule],
    queryFn: () => configApi.getSchema(selectedModule),
    enabled: !!selectedModule,
  });

  const upsertMutation = useMutation({
    mutationFn: () => {
      const parsed = JSON.parse(configText);
      return configApi.upsertConfig(selectedModule, parsed, selectedTenantId);
    },
    onSuccess: () => {
      toast("Configuration saved", "success");
      qc.invalidateQueries({ queryKey: ["config", selectedTenantId, selectedModule] });
      qc.invalidateQueries({ queryKey: ["config-history", selectedTenantId, selectedModule] });
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const tenantOptions = [
    { value: "", label: "Select a tenant…" },
    ...(tenantsQuery.data?.data ?? []).map((t) => ({
      value: t.id,
      label: `${t.name} (${t.slug})`,
    })),
  ];

  const moduleOptions = [
    { value: "", label: "Select a module…" },
    ...(modulesQuery.data ?? []).map((m) => ({
      value: m.module,
      label: m.module,
    })),
  ];

  const isValidJson = (() => {
    try { JSON.parse(configText); return true; } catch { return false; }
  })();

  const canSave =
    !!selectedTenantId &&
    !!selectedModule &&
    isValidJson &&
    !upsertMutation.isPending;

  return (
    <Shell>
      <div className="space-y-5 max-w-4xl">
        <div>
          <h2 className="text-lg font-semibold text-slate-900">Configuration</h2>
          <p className="text-sm text-slate-500 mt-0.5">
            Manage per-tenant module configuration with JSON schema validation.
          </p>
        </div>

        {/* Selectors */}
        <div className="flex gap-3 flex-wrap">
          <div className="w-64">
            <Select
              label="Tenant"
              options={tenantOptions}
              value={selectedTenantId}
              onChange={(e) => { setSelectedTenantId(e.target.value); setShowHistory(false); }}
            />
          </div>
          <div className="w-48">
            <Select
              label="Module"
              options={moduleOptions}
              value={selectedModule}
              onChange={(e) => { setSelectedModule(e.target.value); setShowHistory(false); setShowSchema(false); }}
            />
          </div>
        </div>

        {!selectedTenantId || !selectedModule ? (
          <EmptyState
            icon={Settings2}
            title="Select a tenant and module"
            description="Choose a tenant and module above to view and edit its configuration."
          />
        ) : configQuery.isLoading ? (
          <PageSpinner />
        ) : (
          <div className="space-y-5">
            {/* Config editor */}
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>{selectedModule}</CardTitle>
                    <CardDescription>
                      {configQuery.data?.isDefault
                        ? "Showing schema defaults — no custom config set"
                        : `Last updated ${formatDate(configQuery.data?.updatedAt)}`}
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    {configQuery.data?.isDefault && (
                      <Badge variant="neutral">default</Badge>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        if (configQuery.data) {
                          setConfigText(JSON.stringify(configQuery.data.config, null, 2));
                        }
                      }}
                      title="Reset to current saved config"
                    >
                      <RefreshCw className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                <Textarea
                  value={configText}
                  onChange={(e) => setConfigText(e.target.value)}
                  rows={14}
                  className="font-mono text-xs"
                  error={!isValidJson && configText.length > 0 ? "Invalid JSON" : undefined}
                />
                <div className="flex items-center justify-between">
                  <div className="flex gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowSchema((v) => !v)}
                    >
                      <Code2 className="h-3.5 w-3.5" />
                      {showSchema ? "Hide Schema" : "View Schema"}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowHistory((v) => !v)}
                    >
                      <History className="h-3.5 w-3.5" />
                      {showHistory ? "Hide History" : "View History"}
                    </Button>
                  </div>
                  <Button
                    disabled={!canSave}
                    loading={upsertMutation.isPending}
                    onClick={() => upsertMutation.mutate()}
                  >
                    <Save className="h-4 w-4" />
                    Save Configuration
                  </Button>
                </div>
              </CardContent>
            </Card>

            {/* Schema viewer */}
            {showSchema && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <Code2 className="h-4 w-4 text-slate-400" />
                    <CardTitle>JSON Schema — {selectedModule}</CardTitle>
                  </div>
                </CardHeader>
                <CardContent>
                  {schemaQuery.isLoading ? (
                    <PageSpinner />
                  ) : (
                    <pre className="text-xs font-mono bg-slate-900 text-green-400 rounded-lg p-4 overflow-x-auto scrollbar-thin max-h-72">
                      {JSON.stringify(schemaQuery.data, null, 2)}
                    </pre>
                  )}
                </CardContent>
              </Card>
            )}

            {/* History */}
            {showHistory && (
              <HistoryPanel
                module={selectedModule}
                tenantId={selectedTenantId}
              />
            )}
          </div>
        )}
      </div>
    </Shell>
  );
}
