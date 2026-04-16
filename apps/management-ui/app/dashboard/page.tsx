/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { PageSpinner } from "@/components/ui/spinner";
import { checkAllHealth, type ServiceHealth } from "@/lib/api/health";
import { tenantsApi } from "@/lib/api/tenants";
import {
  Building2,
  Key,
  CalendarDays,
  Settings2,
  ArrowRight,
  Activity,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";

function HealthDot({ status }: { status: ServiceHealth["status"] }) {
  return (
    <span
      className={cn(
        "inline-block h-2 w-2 rounded-full",
        status === "ok"      && "bg-green-500",
        status === "error"   && "bg-red-500",
        status === "loading" && "bg-slate-300 animate-pulse",
      )}
    />
  );
}

function ServiceHealthCard({ health }: { health: ServiceHealth }) {
  return (
    <div className="flex items-center justify-between py-2.5 border-b last:border-0 border-slate-100">
      <div className="flex items-center gap-3">
        <HealthDot status={health.status} />
        <span className="text-sm font-medium text-slate-700">{health.name}</span>
      </div>
      <div className="flex items-center gap-2">
        {health.latencyMs !== undefined && (
          <span className="text-xs text-slate-400">{health.latencyMs}ms</span>
        )}
        {health.status === "ok" ? (
          <CheckCircle2 className="h-4 w-4 text-green-500" />
        ) : (
          <XCircle className="h-4 w-4 text-red-400" />
        )}
      </div>
    </div>
  );
}

const quickLinks = [
  {
    href: "/tenants",
    label: "Manage Tenants",
    description: "Create and configure tenants",
    icon: Building2,
    color: "text-violet-600 bg-violet-50",
  },
  {
    href: "/keys",
    label: "API Keys",
    description: "Issue and revoke API keys",
    icon: Key,
    color: "text-amber-600 bg-amber-50",
  },
  {
    href: "/bookings",
    label: "Bookings",
    description: "View and manage bookings",
    icon: CalendarDays,
    color: "text-blue-600 bg-blue-50",
  },
  {
    href: "/config",
    label: "Configuration",
    description: "Manage module config",
    icon: Settings2,
    color: "text-brand-600 bg-brand-50",
  },
];

export default function DashboardPage() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: checkAllHealth,
    refetchInterval: 30_000,
  });

  const tenantsQuery = useQuery({
    queryKey: ["tenants", "summary"],
    queryFn: () => tenantsApi.list({ limit: 1 }),
  });

  const healthData = healthQuery.data ?? [];
  const healthyCount = healthData.filter((h) => h.status === "ok").length;
  const totalCount = healthData.length;
  const allHealthy = healthyCount === totalCount && totalCount > 0;

  return (
    <Shell>
      <div className="space-y-6 max-w-5xl">
        {/* Page title */}
        <div>
          <h2 className="text-lg font-semibold text-slate-900">Overview</h2>
          <p className="text-sm text-slate-500 mt-0.5">
            ServiceForge platform health and quick actions.
          </p>
        </div>

        {/* Stat cards */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Card>
            <CardContent className="flex items-center gap-4 py-5">
              <div className="rounded-lg bg-violet-50 p-3">
                <Building2 className="h-5 w-5 text-violet-600" />
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">
                  Total Tenants
                </p>
                <p className="text-2xl font-bold text-slate-900">
                  {tenantsQuery.isLoading ? "—" : (tenantsQuery.data?.total ?? "—")}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="flex items-center gap-4 py-5">
              <div className="rounded-lg bg-green-50 p-3">
                <Activity className="h-5 w-5 text-green-600" />
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">
                  Services Online
                </p>
                <p className="text-2xl font-bold text-slate-900">
                  {healthQuery.isLoading ? "—" : `${healthyCount}/${totalCount}`}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="flex items-center gap-4 py-5">
              <div
                className={cn(
                  "rounded-lg p-3",
                  allHealthy ? "bg-green-50" : "bg-red-50",
                )}
              >
                {allHealthy ? (
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                ) : (
                  <XCircle className="h-5 w-5 text-red-500" />
                )}
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500 uppercase tracking-wide">
                  Platform Status
                </p>
                <p className="text-sm font-semibold text-slate-900 mt-0.5">
                  {healthQuery.isLoading
                    ? "Checking…"
                    : allHealthy
                    ? "All systems go"
                    : "Degraded"}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Service health */}
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle>Service Health</CardTitle>
                <Badge variant={allHealthy ? "success" : "error"}>
                  {allHealthy ? "Healthy" : "Issues"}
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              {healthQuery.isLoading ? (
                <PageSpinner />
              ) : (
                <div>
                  {healthData.map((h) => (
                    <ServiceHealthCard key={h.name} health={h} />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Quick actions */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle>Quick Actions</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 pt-2">
              {quickLinks.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  className="flex items-center gap-4 rounded-lg p-3 hover:bg-slate-50 transition-colors group"
                >
                  <div className={cn("rounded-lg p-2", link.color)}>
                    <link.icon className="h-4 w-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-slate-900">{link.label}</p>
                    <p className="text-xs text-slate-500">{link.description}</p>
                  </div>
                  <ArrowRight className="h-4 w-4 text-slate-300 group-hover:text-slate-500 transition-colors" />
                </Link>
              ))}
            </CardContent>
          </Card>
        </div>
      </div>
    </Shell>
  );
}
