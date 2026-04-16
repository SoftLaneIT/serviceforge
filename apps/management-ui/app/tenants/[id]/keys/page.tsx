/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { use, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { PageSpinner } from "@/components/ui/spinner";
import { KeysTable } from "@/components/keys/keys-table";
import { IssueKeyDialog } from "@/components/keys/issue-key-dialog";
import { keysApi } from "@/lib/api/keys";
import { tenantsApi } from "@/lib/api/tenants";
import { ArrowLeft, Plus } from "lucide-react";

export default function TenantKeysPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [issueOpen, setIssueOpen] = useState(false);
  const queryKey = ["keys", id];

  const tenantQuery = useQuery({
    queryKey: ["tenant", id],
    queryFn: () => tenantsApi.get(id),
  });

  const keysQuery = useQuery({
    queryKey,
    queryFn: () => keysApi.list(id),
  });

  return (
    <Shell>
      <div className="space-y-5 max-w-5xl">
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
            <h2 className="text-lg font-semibold text-slate-900">API Keys</h2>
            <p className="text-sm text-slate-500 mt-0.5">
              Keys for{" "}
              <span className="font-medium">
                {tenantQuery.data?.name ?? id}
              </span>
            </p>
          </div>
          <Button onClick={() => setIssueOpen(true)}>
            <Plus className="h-4 w-4" />
            Issue Key
          </Button>
        </div>

        <Card>
          <CardContent className="p-0">
            {keysQuery.isLoading ? (
              <PageSpinner />
            ) : (
              <KeysTable
                keys={keysQuery.data ?? []}
                tenantId={id}
                queryKey={queryKey}
              />
            )}
          </CardContent>
        </Card>
      </div>

      <IssueKeyDialog
        open={issueOpen}
        onClose={() => setIssueOpen(false)}
        tenantId={id}
        queryKey={queryKey}
      />
    </Shell>
  );
}
