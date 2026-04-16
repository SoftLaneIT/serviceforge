"use client";

import { useState, useRef, useEffect } from "react";
import { usePathname } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { useTenant } from "@/lib/context/tenant-context";
import { tenantsApi, type Tenant } from "@/lib/api/tenants";
import {
  Building2,
  ChevronDown,
  Check,
  SearchIcon,
  Layers,
} from "lucide-react";

const labels: Record<string, string> = {
  "/dashboard":    "Dashboard",
  "/tenants":      "Tenants",
  "/keys":         "API Keys",
  "/bookings":     "Bookings",
  "/config":       "Configuration",
  "/integrations": "Integrations",
};

function resolveTitle(pathname: string): string {
  const match = Object.keys(labels)
    .sort((a, b) => b.length - a.length)
    .find((k) => pathname === k || pathname.startsWith(k + "/"));
  return match ? labels[match] : "ServiceForge";
}

// 
// Tenant picker dropdown
// 

function TenantSwitcher() {
  const { activeTenant, setActiveTenant } = useTenant();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const ref = useRef<HTMLDivElement>(null);

  const tenantsQuery = useQuery({
    queryKey: ["tenants", "active-all"],
    queryFn: () => tenantsApi.list({ status: "active", limit: 200 }),
    staleTime: 60_000,
  });

  const tenants: Tenant[] = tenantsQuery.data?.data ?? [];

  const filtered = tenants.filter(
    (t) =>
      !search ||
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.slug.toLowerCase().includes(search.toLowerCase()),
  );

  // Close on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
        setSearch("");
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const select = (tenant: Tenant) => {
    setActiveTenant(tenant);
    setOpen(false);
    setSearch("");
  };

  const clear = () => {
    setActiveTenant(null);
    setOpen(false);
    setSearch("");
  };

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm transition-colors",
          "hover:bg-slate-50 focus:outline-none",
          activeTenant
            ? "border-brand-200 bg-brand-50 text-brand-800"
            : "border-slate-200 bg-white text-slate-600",
        )}
      >
        <Building2 className={cn("h-3.5 w-3.5 shrink-0", activeTenant ? "text-brand-500" : "text-slate-400")} />
        <span className="max-w-[180px] truncate font-medium">
          {activeTenant ? activeTenant.name : "All Tenants"}
        </span>
        {activeTenant && (
          <span className="text-xs font-mono text-brand-500 opacity-70">
            ({activeTenant.slug})
          </span>
        )}
        <ChevronDown className={cn("h-3.5 w-3.5 text-slate-400 transition-transform", open && "rotate-180")} />
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1.5 w-72 rounded-xl border border-slate-200 bg-white shadow-lg z-50 overflow-hidden">
          {/* Search */}
          <div className="flex items-center gap-2 border-b border-slate-100 px-3 py-2">
            <SearchIcon className="h-3.5 w-3.5 text-slate-400 shrink-0" />
            <input
              autoFocus
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search tenants…"
              className="flex-1 text-sm bg-transparent focus:outline-none placeholder:text-slate-400"
            />
          </div>

          <div className="max-h-60 overflow-y-auto">
            {/* All tenants option */}
            <button
              onClick={clear}
              className={cn(
                "w-full flex items-center gap-3 px-3 py-2.5 text-sm hover:bg-slate-50 transition-colors",
                !activeTenant && "bg-brand-50",
              )}
            >
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-slate-100 shrink-0">
                <Layers className="h-3.5 w-3.5 text-slate-500" />
              </div>
              <div className="flex-1 text-left">
                <p className="font-medium text-slate-700">All Tenants</p>
                <p className="text-xs text-slate-400">Global view</p>
              </div>
              {!activeTenant && <Check className="h-3.5 w-3.5 text-brand-500" />}
            </button>

            {filtered.length === 0 && search && (
              <p className="px-3 py-4 text-xs text-center text-slate-400">No tenants found.</p>
            )}

            {filtered.map((t) => {
              const active = activeTenant?.id === t.id;
              const initials = t.name.split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase();
              return (
                <button
                  key={t.id}
                  onClick={() => select(t)}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2.5 text-sm hover:bg-slate-50 transition-colors",
                    active && "bg-brand-50",
                  )}
                >
                  <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-brand-100 shrink-0">
                    <span className="text-[11px] font-bold text-brand-700">{initials}</span>
                  </div>
                  <div className="flex-1 text-left min-w-0">
                    <p className="font-medium text-slate-800 truncate">{t.name}</p>
                    <p className="text-xs text-slate-400 font-mono">{t.slug}</p>
                  </div>
                  {active && <Check className="h-3.5 w-3.5 text-brand-500 shrink-0" />}
                </button>
              );
            })}
          </div>

          <div className="border-t border-slate-100 px-3 py-2">
            <p className="text-xs text-slate-400">{tenants.length} active tenant{tenants.length !== 1 ? "s" : ""}</p>
          </div>
        </div>
      )}
    </div>
  );
}

// 
// Header
// 

export function Header() {
  const pathname = usePathname();
  const title = resolveTitle(pathname);

  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-6 gap-4">
      <h1 className="text-sm font-semibold text-slate-900 shrink-0">{title}</h1>

      <div className="flex items-center gap-3 ml-auto">
        <TenantSwitcher />
        <div className="h-4 w-px bg-slate-200" />
        <span className="rounded-full bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-700">
          Admin
        </span>
      </div>
    </header>
  );
}
