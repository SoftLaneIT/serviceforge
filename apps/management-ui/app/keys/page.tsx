/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { PageSpinner } from "@/components/ui/spinner";
import { KeysTable } from "@/components/keys/keys-table";
import { IssueKeyDialog } from "@/components/keys/issue-key-dialog";
import { tenantsApi } from "@/lib/api/tenants";
import { keysApi } from "@/lib/api/keys";
import { Plus } from "lucide-react";

export default function KeysPage() {
  const [issueOpen, setIssueOpen] = useState(false);
  const [selectedTenantId, setSelectedTenantId] = useState("");

  const tenantsQuery = useQuery({
    queryKey: ["tenants", "all"],
    queryFn: () => tenantsApi.list({ status: "active", limit: 200 }),
  });

  const tenantOptions = [
    { value: "", label: "Select a tenant…" },
    ...(tenantsQuery.data?.tenants ?? []).map((t) => ({
      value: t.id,
      label: `${t.name} (${t.slug})`,
    })),
  ];

  const keysQueryKey = ["keys", selectedTenantId];
  const keysQuery = useQuery({
    queryKey: keysQueryKey,
    queryFn: () => keysApi.list(selectedTenantId),
    enabled: !!selectedTenantId,
  });

  return (
    <Shell>
      <div className="space-y-5 max-w-5xl">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">API Keys</h2>
            <p className="text-sm text-slate-500 mt-0.5">
              Issue and manage API keys per tenant.
            </p>
          </div>
          <Button
            disabled={!selectedTenantId}
            onClick={() => setIssueOpen(true)}
          >
            <Plus className="h-4 w-4" />
            Issue Key
          </Button>
        </div>

        {/* Tenant selector */}
        <div className="w-72">
          <Select
            label="Tenant"
            options={tenantOptions}
            value={selectedTenantId}
            onChange={(e) => setSelectedTenantId(e.target.value)}
          />
        </div>

        <Card>
          <CardContent className="p-0">
            {!selectedTenantId ? (
              <div className="py-16 text-center text-sm text-slate-500">
                Select a tenant to view its API keys.
              </div>
            ) : keysQuery.isLoading ? (
              <PageSpinner />
            ) : (
              <KeysTable
                keys={keysQuery.data ?? []}
                tenantId={selectedTenantId}
                queryKey={keysQueryKey}
              />
            )}
          </CardContent>
        </Card>
      </div>

      {selectedTenantId && (
        <IssueKeyDialog
          open={issueOpen}
          onClose={() => setIssueOpen(false)}
          tenantId={selectedTenantId}
          queryKey={keysQueryKey}
        />
      )}
    </Shell>
  );
}
