"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  Building2,
  Key,
  CalendarDays,
  Settings2,
  Zap,
  ChevronRight,
  Plug,
} from "lucide-react";

interface NavItem {
  href: string;
  label: string;
  icon: React.ElementType;
  exact?: boolean;
}

const navItems: NavItem[] = [
  { href: "/dashboard",    label: "Dashboard",    icon: LayoutDashboard, exact: true },
  { href: "/tenants",      label: "Tenants",      icon: Building2 },
  { href: "/keys",         label: "API Keys",     icon: Key },
  { href: "/bookings",     label: "Bookings",     icon: CalendarDays },
  { href: "/config",       label: "Config",       icon: Settings2 },
  { href: "/integrations", label: "Integrations", icon: Plug },
];

export function Sidebar() {
  const pathname = usePathname();

  const isActive = (item: NavItem) =>
    item.exact ? pathname === item.href : pathname.startsWith(item.href);

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-slate-200 bg-white">
      {/* Logo */}
      <div className="flex items-center gap-2.5 px-5 py-4 border-b border-slate-100">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600">
          <Zap className="h-4 w-4 text-white" />
        </div>
        <div>
          <span className="text-sm font-bold text-slate-900">ServiceForge</span>
          <span className="block text-xs text-slate-400 leading-none">Management</span>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-0.5">
        {navItems.map((item) => {
          const active = isActive(item);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                active
                  ? "bg-brand-50 text-brand-700"
                  : "text-slate-600 hover:bg-slate-50 hover:text-slate-900",
              )}
            >
              <item.icon
                className={cn("h-4 w-4 shrink-0", active ? "text-brand-600" : "text-slate-400")}
              />
              <span className="flex-1">{item.label}</span>
              {active && <ChevronRight className="h-3.5 w-3.5 text-brand-400" />}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="border-t border-slate-100 px-5 py-4">
        <p className="text-xs text-slate-400">v0.1.0 · SoftlaneIT</p>
      </div>
    </aside>
  );
}
