/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { use, useEffect, Suspense } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import Link from "next/link";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/input";
import { TenantStatusBadge, PlanBadge } from "@/components/ui/badge";
import { PageSpinner } from "@/components/ui/spinner";
import { useToast } from "@/components/ui/toast";
import { tenantsApi, type TenantPlan } from "@/lib/api/tenants";
import { formatDate } from "@/lib/utils";
import { Key, CalendarDays, Settings2, ArrowLeft, Save } from "lucide-react";

const updateSchema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  plan: z.enum(["starter", "pro", "enterprise"] as const),
  settings: z.string().refine((v) => {
    try { JSON.parse(v); return true; } catch { return false; }
  }, "Must be valid JSON"),
});
type UpdateForm = z.infer<typeof updateSchema>;

const planOptions = [
  { value: "starter", label: "Starter" },
  { value: "pro", label: "Pro" },
  { value: "enterprise", label: "Enterprise" },
];

const subNavLinks = (id: string) => [
  { href: `/tenants/${id}/keys`,     label: "API Keys",     icon: Key },
  { href: `/tenants/${id}/bookings`, label: "Bookings",     icon: CalendarDays },
  { href: `/tenants/${id}/config`,   label: "Config",       icon: Settings2 },
];

// Inner component — safe to call use(params) here because it's wrapped in Suspense
function TenantDetailContent({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { toast } = useToast();
  const qc = useQueryClient();

  const { data: tenant, isLoading } = useQuery({
    queryKey: ["tenant", id],
    queryFn: () => tenantsApi.get(id),
  });

  const updateMutation = useMutation({
    mutationFn: (body: UpdateForm) =>
      tenantsApi.update(id, {
        name: body.name,
        plan: body.plan as TenantPlan,
        settings: JSON.parse(body.settings),
      }),
    onSuccess: () => {
      toast("Tenant updated", "success");
      qc.invalidateQueries({ queryKey: ["tenant", id] });
      qc.invalidateQueries({ queryKey: ["tenants"] });
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<UpdateForm>({
    resolver: zodResolver(updateSchema),
  });

  useEffect(() => {
    if (tenant) {
      reset({
        name: tenant.name,
        plan: tenant.plan,
        settings: JSON.stringify(tenant.settings ?? {}, null, 2),
      });
    }
  }, [tenant, reset]);

  return (
    <div className="space-y-6 max-w-2xl">
      <div className="flex items-center gap-3">
        <Link href="/tenants">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4" />
            Tenants
          </Button>
        </Link>
      </div>

      {isLoading ? (
        <PageSpinner />
      ) : !tenant ? (
        <p className="text-sm text-slate-500">Tenant not found.</p>
      ) : (
        <>
          {/* Header */}
          <div className="flex items-start justify-between">
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold text-slate-900">{tenant.name}</h2>
                <TenantStatusBadge status={tenant.status} />
                <PlanBadge plan={tenant.plan} />
              </div>
              <p className="text-sm text-slate-500 mt-0.5 font-mono">{tenant.slug}</p>
            </div>
          </div>

          {/* Sub-nav quick links */}
          <div className="flex gap-2 flex-wrap">
            {subNavLinks(id).map((l) => (
              <Link key={l.href} href={l.href}>
                <Button variant="outline" size="sm">
                  <l.icon className="h-3.5 w-3.5" />
                  {l.label}
                </Button>
              </Link>
            ))}
          </div>

          {/* Meta */}
          <Card>
            <CardHeader>
              <CardTitle>Details</CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">ID</p>
                <p className="font-mono text-xs mt-1 text-slate-700">{tenant.id}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">Created</p>
                <p className="mt-1 text-slate-700">{formatDate(tenant.createdAt)}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">Updated</p>
                <p className="mt-1 text-slate-700">{formatDate(tenant.updatedAt)}</p>
              </div>
            </CardContent>
          </Card>

          {/* Edit form */}
          <Card>
            <CardHeader>
              <CardTitle>Edit Tenant</CardTitle>
              <CardDescription>Update name, plan, or settings.</CardDescription>
            </CardHeader>
            <CardContent>
              <form
                onSubmit={handleSubmit((v) => updateMutation.mutate(v))}
                className="space-y-4"
              >
                <Input
                  label="Name"
                  error={errors.name?.message}
                  {...register("name")}
                />
                <Select
                  label="Plan"
                  options={planOptions}
                  error={errors.plan?.message}
                  {...register("plan")}
                />
                <Textarea
                  label="Settings (JSON)"
                  rows={5}
                  className="font-mono text-xs"
                  hint="Free-form JSON metadata for this tenant"
                  error={errors.settings?.message}
                  {...register("settings")}
                />
                <div className="flex justify-end">
                  <Button type="submit" loading={updateMutation.isPending}>
                    <Save className="h-4 w-4" />
                    Save Changes
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

// Outer page wraps the content in Suspense so use(params) has a boundary above it
export default function TenantDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  return (
    <Shell>
      <Suspense fallback={<PageSpinner />}>
        <TenantDetailContent params={params} />
      </Suspense>
    </Shell>
  );
}
