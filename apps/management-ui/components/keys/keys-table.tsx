"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { KeyStatusBadge, EnvBadge } from "@/components/ui/badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { useToast } from "@/components/ui/toast";
import { keysApi, type ApiKey } from "@/lib/api/keys";
import { formatDate } from "@/lib/utils";
import { Key, Trash2, Copy, Check } from "lucide-react";

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <button
      onClick={copy}
      className="inline-flex items-center gap-1 text-xs text-slate-400 hover:text-slate-700 transition-colors"
      title="Copy"
    >
      {copied ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
    </button>
  );
}

interface KeysTableProps {
  keys: ApiKey[];
  tenantId: string;
  queryKey: unknown[];
}

export function KeysTable({ keys, tenantId, queryKey }: KeysTableProps) {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [revokeTarget, setRevokeTarget] = useState<ApiKey | null>(null);

  const revokeMutation = useMutation({
    mutationFn: (id: string) => keysApi.revoke(id, tenantId),
    onSuccess: () => {
      toast("API key revoked", "success");
      qc.invalidateQueries({ queryKey });
      setRevokeTarget(null);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  if (keys.length === 0) {
    return (
      <EmptyState
        icon={Key}
        title="No API keys"
        description="Issue an API key to allow tenants to authenticate with ServiceForge."
      />
    );
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Prefix</TableHead>
            <TableHead>Environment</TableHead>
            <TableHead>Scope</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Last Used</TableHead>
            <TableHead>Expires</TableHead>
            <TableHead className="w-16" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {keys.map((k) => (
            <TableRow key={k.id}>
              <TableCell className="font-medium text-slate-900">{k.name}</TableCell>
              <TableCell>
                <div className="flex items-center gap-1.5 font-mono text-xs text-slate-600">
                  {k.keyPrefix}…
                  <CopyButton value={k.keyPrefix} />
                </div>
              </TableCell>
              <TableCell><EnvBadge env={k.environment} /></TableCell>
              <TableCell className="text-xs text-slate-500">
                {k.moduleScope.length === 0 ? "all modules" : k.moduleScope.join(", ")}
              </TableCell>
              <TableCell><KeyStatusBadge status={k.status} /></TableCell>
              <TableCell className="text-slate-500 text-xs">{formatDate(k.lastUsedAt)}</TableCell>
              <TableCell className="text-slate-500 text-xs">{formatDate(k.expiresAt)}</TableCell>
              <TableCell>
                {k.status === "active" && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-red-500 hover:bg-red-50"
                    onClick={() => setRevokeTarget(k)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <ConfirmDialog
        open={!!revokeTarget}
        onClose={() => setRevokeTarget(null)}
        onConfirm={() => revokeTarget && revokeMutation.mutate(revokeTarget.id)}
        title="Revoke API Key"
        description={`Revoke "${revokeTarget?.name}"? Any request using this key will immediately be rejected.`}
        confirmLabel="Revoke Key"
        destructive
        loading={revokeMutation.isPending}
      />
    </>
  );
}
