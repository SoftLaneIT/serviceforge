"use client";

import { usePathname } from "next/navigation";

const labels: Record<string, string> = {
  "/dashboard": "Dashboard",
  "/tenants":   "Tenants",
  "/keys":      "API Keys",
  "/bookings":  "Bookings",
  "/config":    "Configuration",
};

function resolveTitle(pathname: string): string {
  const match = Object.keys(labels)
    .sort((a, b) => b.length - a.length)
    .find((k) => pathname === k || pathname.startsWith(k + "/"));
  return match ? labels[match] : "ServiceForge";
}

export function Header() {
  const pathname = usePathname();
  const title = resolveTitle(pathname);

  return (
    <header className="flex h-14 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-6">
      <h1 className="text-sm font-semibold text-slate-900">{title}</h1>
      <div className="flex items-center gap-3">
        <span className="rounded-full bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-700">
          Admin
        </span>
      </div>
    </header>
  );
}
