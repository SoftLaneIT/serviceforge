/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import Link from "next/link";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Dialog } from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from "@/components/ui/table";
import { TenantStatusBadge, PlanBadge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { PageSpinner } from "@/components/ui/spinner";
import { useToast } from "@/components/ui/toast";
import { tenantsApi, type Tenant, type TenantPlan } from "@/lib/api/tenants";
import { formatDate } from "@/lib/utils";
import { Building2, Plus, Trash2, Eye } from "lucide-react";

const createSchema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  slug: z
    .string()
    .min(2)
    .regex(/^[a-z0-9-]+$/, "Slug may only contain lowercase letters, numbers and hyphens"),
  plan: z.enum(["starter", "pro", "enterprise"] as const),
});
type CreateForm = z.infer<typeof createSchema>;

const planOptions = [
  { value: "starter", label: "Starter" },
  { value: "pro", label: "Pro" },
  { value: "enterprise", label: "Enterprise" },
];

const statusOptions = [
  { value: "", label: "All statuses" },
  { value: "active", label: "Active" },
  { value: "suspended", label: "Suspended" },
  { value: "deleted", label: "Deleted" },
];

export default function TenantsPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Tenant | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [page, setPage] = useState(0);
  const limit = 20;

  const { data, isLoading } = useQuery({
    queryKey: ["tenants", statusFilter, page],
    queryFn: () =>
      tenantsApi.list({
        status: statusFilter as Tenant["status"] | undefined || undefined,
        limit,
        offset: page * limit,
      }),
  });

  const createMutation = useMutation({
    mutationFn: (body: CreateForm) =>
      tenantsApi.create({ ...body, plan: body.plan as TenantPlan }),
    onSuccess: () => {
      toast("Tenant created successfully", "success");
      qc.invalidateQueries({ queryKey: ["tenants"] });
      setCreateOpen(false);
      reset();
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => tenantsApi.delete(id),
    onSuccess: () => {
      toast("Tenant deleted", "success");
      qc.invalidateQueries({ queryKey: ["tenants"] });
      setDeleteTarget(null);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateForm>({
    resolver: zodResolver(createSchema),
    defaultValues: { plan: "starter" },
  });

  const tenants = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / limit);

  return (
    <Shell>
      <div className="space-y-5 max-w-5xl">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Tenants</h2>
            <p className="text-sm text-slate-500 mt-0.5">
              {total} tenant{total !== 1 ? "s" : ""} registered
            </p>
          </div>
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            New Tenant
          </Button>
        </div>

        <Card>
          {/* Filters */}
          <div className="px-5 py-3 border-b border-slate-100 flex items-center gap-3">
            <Select
              options={statusOptions}
              value={statusFilter}
              onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}
              className="w-40"
            />
          </div>

          <CardContent className="p-0">
            {isLoading ? (
              <PageSpinner />
            ) : tenants.length === 0 ? (
              <EmptyState
                icon={Building2}
                title="No tenants yet"
                description="Create your first tenant to get started."
                action={
                  <Button onClick={() => setCreateOpen(true)}>
                    <Plus className="h-4 w-4" />
                    New Tenant
                  </Button>
                }
              />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Slug</TableHead>
                    <TableHead>Plan</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="w-24" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tenants.map((t) => (
                    <TableRow key={t.id}>
                      <TableCell className="font-medium text-slate-900">{t.name}</TableCell>
                      <TableCell className="font-mono text-xs text-slate-500">{t.slug}</TableCell>
                      <TableCell><PlanBadge plan={t.plan} /></TableCell>
                      <TableCell><TenantStatusBadge status={t.status} /></TableCell>
                      <TableCell className="text-slate-500">{formatDate(t.createdAt)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Link href={`/tenants/${t.id}`}>
                            <Button variant="ghost" size="sm" aria-label="View">
                              <Eye className="h-3.5 w-3.5" />
                            </Button>
                          </Link>
                          <Button
                            variant="ghost"
                            size="sm"
                            aria-label="Delete"
                            className="text-red-500 hover:bg-red-50"
                            onClick={() => setDeleteTarget(t)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between px-5 py-3 border-t border-slate-100">
              <span className="text-xs text-slate-500">
                Page {page + 1} of {totalPages}
              </span>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page + 1 >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </Card>
      </div>

      {/* Create dialog */}
      <Dialog
        open={createOpen}
        onClose={() => { setCreateOpen(false); reset(); }}
        title="Create Tenant"
        description="Add a new tenant to ServiceForge."
      >
        <form
          onSubmit={handleSubmit((v) => createMutation.mutate(v))}
          className="space-y-4"
        >
          <Input
            label="Name"
            placeholder="Acme Corp"
            error={errors.name?.message}
            {...register("name")}
          />
          <Input
            label="Slug"
            placeholder="acme-corp"
            hint="Unique identifier — lowercase letters, numbers, hyphens"
            error={errors.slug?.message}
            {...register("slug")}
          />
          <Select
            label="Plan"
            options={planOptions}
            error={errors.plan?.message}
            {...register("plan")}
          />
          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => { setCreateOpen(false); reset(); }}
            >
              Cancel
            </Button>
            <Button type="submit" loading={createMutation.isPending}>
              Create Tenant
            </Button>
          </div>
        </form>
      </Dialog>

      {/* Delete confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title="Delete Tenant"
        description={`Are you sure you want to delete "${deleteTarget?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        destructive
        loading={deleteMutation.isPending}
      />
    </Shell>
  );
}
