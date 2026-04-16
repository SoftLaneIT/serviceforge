"use client";

import { useState, useEffect, useRef } from "react";
import Link from "next/link";
import {
  Zap,
  Copy,
  Check,
  ChevronRight,
  BookOpen,
  Code2,
  Key,
  Calendar,
  Settings,
  Webhook,
  AlertCircle,
  Terminal,
  Globe,
  Shield,
  Clock,
  Bell,
  Palette,
  Users,
  LayoutDashboard,
  Menu,
  X,
} from "lucide-react";

// ─── helpers ─────────────────────────────────────────────────────────────────

function useClipboard(timeout = 1800) {
  const [copied, setCopied] = useState<string | null>(null);
  const copy = (id: string, text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(id);
      setTimeout(() => setCopied(null), timeout);
    });
  };
  return { copied, copy };
}

// ─── components ──────────────────────────────────────────────────────────────

function CodeBlock({
  id,
  code,
  lang = "bash",
  copied,
  onCopy,
}: {
  id: string;
  code: string;
  lang?: string;
  copied: string | null;
  onCopy: (id: string, text: string) => void;
}) {
  return (
    <div className="relative my-4 rounded-xl overflow-hidden border border-slate-700 bg-slate-900">
      <div className="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700">
        <span className="text-xs font-mono text-slate-400">{lang}</span>
        <button
          onClick={() => onCopy(id, code)}
          className="flex items-center gap-1.5 rounded px-2 py-1 text-xs text-slate-400 hover:text-white hover:bg-slate-700 transition-colors"
        >
          {copied === id ? (
            <><Check className="h-3 w-3 text-green-400" /><span className="text-green-400">Copied!</span></>
          ) : (
            <><Copy className="h-3 w-3" /><span>Copy</span></>
          )}
        </button>
      </div>
      <pre className="p-4 overflow-x-auto text-sm text-slate-200 font-mono leading-relaxed whitespace-pre">
        {code}
      </pre>
    </div>
  );
}

function Badge({ children, color = "blue" }: { children: React.ReactNode; color?: "blue" | "green" | "yellow" | "red" | "purple" | "slate" }) {
  const colors = {
    blue:   "bg-blue-100 text-blue-700 border border-blue-200",
    green:  "bg-green-100 text-green-700 border border-green-200",
    yellow: "bg-yellow-100 text-yellow-700 border border-yellow-200",
    red:    "bg-red-100 text-red-700 border border-red-200",
    purple: "bg-purple-100 text-purple-700 border border-purple-200",
    slate:  "bg-slate-100 text-slate-600 border border-slate-200",
  };
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${colors[color]}`}>
      {children}
    </span>
  );
}

function MethodBadge({ method }: { method: string }) {
  const colors: Record<string, string> = {
    GET:    "bg-blue-600",
    POST:   "bg-green-600",
    PUT:    "bg-yellow-600",
    PATCH:  "bg-orange-600",
    DELETE: "bg-red-600",
  };
  return (
    <span className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-bold text-white font-mono ${colors[method] ?? "bg-slate-600"}`}>
      {method}
    </span>
  );
}

function EndpointRow({ method, path, desc }: { method: string; path: string; desc: string }) {
  return (
    <div className="flex items-start gap-3 py-3 border-b border-slate-100 last:border-0">
      <MethodBadge method={method} />
      <code className="text-sm font-mono text-slate-700 flex-1 min-w-0">{path}</code>
      <span className="text-sm text-slate-500 text-right hidden sm:block">{desc}</span>
    </div>
  );
}

function Section({
  id,
  icon: Icon,
  title,
  children,
}: {
  id: string;
  icon: React.ElementType;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section id={id} className="scroll-mt-20 mb-16">
      <div className="flex items-center gap-3 mb-6">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-600 shadow-sm flex-shrink-0">
          <Icon className="h-5 w-5 text-white" />
        </div>
        <h2 className="text-2xl font-bold text-slate-900">{title}</h2>
      </div>
      <div className="space-y-4 text-slate-700 leading-relaxed">{children}</div>
    </section>
  );
}

function SubSection({ title, children }: { children: React.ReactNode; title: string }) {
  return (
    <div className="mt-8">
      <h3 className="text-lg font-semibold text-slate-800 mb-3 pb-2 border-b border-slate-200">{title}</h3>
      <div className="space-y-3">{children}</div>
    </div>
  );
}

function ParamTable({ rows }: { rows: { name: string; type: string; required?: boolean; desc: string }[] }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-slate-200 mt-3">
      <table className="w-full text-sm">
        <thead className="bg-slate-50">
          <tr>
            <th className="px-4 py-2.5 text-left font-semibold text-slate-600">Field</th>
            <th className="px-4 py-2.5 text-left font-semibold text-slate-600">Type</th>
            <th className="px-4 py-2.5 text-left font-semibold text-slate-600">Required</th>
            <th className="px-4 py-2.5 text-left font-semibold text-slate-600">Description</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {rows.map((r) => (
            <tr key={r.name} className="hover:bg-slate-50/50">
              <td className="px-4 py-2.5 font-mono text-slate-800 text-xs">{r.name}</td>
              <td className="px-4 py-2.5"><Badge color="purple">{r.type}</Badge></td>
              <td className="px-4 py-2.5">
                {r.required ? <Badge color="red">required</Badge> : <Badge color="slate">optional</Badge>}
              </td>
              <td className="px-4 py-2.5 text-slate-600">{r.desc}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Note({ children, type = "info" }: { children: React.ReactNode; type?: "info" | "warn" | "tip" }) {
  const styles = {
    info: "bg-blue-50 border-blue-200 text-blue-800",
    warn: "bg-yellow-50 border-yellow-200 text-yellow-800",
    tip:  "bg-green-50 border-green-200 text-green-800",
  };
  const icons = { info: AlertCircle, warn: AlertCircle, tip: Check };
  const Icon = icons[type];
  return (
    <div className={`flex items-start gap-3 rounded-xl border p-4 mt-4 ${styles[type]}`}>
      <Icon className="h-4 w-4 mt-0.5 flex-shrink-0" />
      <div className="text-sm">{children}</div>
    </div>
  );
}

// ─── nav config ──────────────────────────────────────────────────────────────

const NAV = [
  { id: "overview",        label: "Overview",              icon: BookOpen },
  { id: "quickstart",      label: "Quick Start",           icon: Zap },
  { id: "authentication",  label: "Authentication",        icon: Key },
  { id: "tenants-api",     label: "Tenants API",           icon: Users },
  { id: "bookings-api",    label: "Bookings API",          icon: Calendar },
  { id: "config-api",      label: "Config API",            icon: Settings },
  { id: "booking-policy",  label: "Booking Policy",        icon: Shield },
  { id: "modules",         label: "Config Modules",        icon: LayoutDashboard },
  { id: "webhooks",        label: "Webhooks",              icon: Webhook },
  { id: "typescript",      label: "TypeScript SDK",        icon: Code2 },
  { id: "errors",          label: "Error Reference",       icon: AlertCircle },
  { id: "docker",          label: "Self-hosting",          icon: Terminal },
];

// ─── page ─────────────────────────────────────────────────────────────────────

export default function DocsPage() {
  const { copied, copy } = useClipboard();
  const [activeId, setActiveId] = useState("overview");
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  // intersection observer for active section
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) setActiveId(e.target.id);
        }
      },
      { rootMargin: "-20% 0px -70% 0px" }
    );
    NAV.forEach(({ id }) => {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    });
    return () => observer.disconnect();
  }, []);

  const C = (id: string, code: string, lang?: string) => (
    <CodeBlock id={id} code={code} lang={lang} copied={copied} onCopy={copy} />
  );

  return (
    <div className="min-h-screen bg-white font-sans">
      {/* ── top bar ── */}
      <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/95 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-3">
            <button
              className="lg:hidden p-1.5 rounded-lg hover:bg-slate-100"
              onClick={() => setMobileNavOpen(!mobileNavOpen)}
            >
              {mobileNavOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
            <Link href="/" className="flex items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-brand-600">
                <Zap className="h-4 w-4 text-white" />
              </div>
              <span className="font-bold text-slate-900">ServiceForge</span>
            </Link>
            <ChevronRight className="h-4 w-4 text-slate-300" />
            <span className="text-sm text-slate-500 font-medium">Developer Docs</span>
          </div>
          <div className="flex items-center gap-3">
            <Badge color="blue">v0.1.0</Badge>
            <Link href="/dashboard" className="hidden sm:flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700 transition-colors">
              <LayoutDashboard className="h-3.5 w-3.5" />
              Dashboard
            </Link>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-7xl flex">
        {/* ── sidebar nav (desktop) ── */}
        <aside className="hidden lg:block sticky top-14 self-start w-60 shrink-0 h-[calc(100vh-3.5rem)] overflow-y-auto py-6 pr-4 pl-2 border-r border-slate-100">
          <p className="px-3 mb-2 text-[10px] font-semibold uppercase tracking-widest text-slate-400">Contents</p>
          <nav className="space-y-0.5">
            {NAV.map(({ id, label, icon: Icon }) => (
              <a
                key={id}
                href={`#${id}`}
                className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors ${
                  activeId === id
                    ? "bg-brand-50 text-brand-700 font-medium"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                }`}
                onClick={() => setMobileNavOpen(false)}
              >
                <Icon className={`h-3.5 w-3.5 shrink-0 ${activeId === id ? "text-brand-600" : "text-slate-400"}`} />
                {label}
              </a>
            ))}
          </nav>
        </aside>

        {/* ── mobile nav overlay ── */}
        {mobileNavOpen && (
          <div className="fixed inset-0 z-30 lg:hidden">
            <div className="absolute inset-0 bg-black/40" onClick={() => setMobileNavOpen(false)} />
            <div className="absolute top-14 left-0 bottom-0 w-64 bg-white shadow-xl overflow-y-auto py-4 px-3">
              {NAV.map(({ id, label, icon: Icon }) => (
                <a
                  key={id}
                  href={`#${id}`}
                  className="flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm text-slate-700 hover:bg-slate-50"
                  onClick={() => setMobileNavOpen(false)}
                >
                  <Icon className="h-4 w-4 text-slate-400" />
                  {label}
                </a>
              ))}
            </div>
          </div>
        )}

        {/* ── main content ── */}
        <main className="flex-1 min-w-0 px-6 lg:px-10 py-10 max-w-4xl">

          {/* ═══════════ OVERVIEW ═══════════ */}
          <Section id="overview" icon={BookOpen} title="Overview">
            <p>
              <strong>ServiceForge</strong> is an open-source, multi-tenant service-capability platform. It gives every
              tenant their own isolated booking engine, API keys, and rich runtime configuration — all managed through a
              single control plane.
            </p>

            <SubSection title="Architecture">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-2">
                {[
                  { icon: Globe, name: "management-service", port: "8081", desc: "Tenant & API-key CRUD (REST)" },
                  { icon: Calendar, name: "booking-service", port: "8084", desc: "Booking lifecycle + policy enforcement" },
                  { icon: Settings, name: "config-service", port: "8085", desc: "JSON-Schema-driven runtime config" },
                  { icon: Terminal, name: "management-ui", port: "3000", desc: "Next.js 14 control-plane dashboard" },
                ].map((s) => (
                  <div key={s.name} className="rounded-xl border border-slate-200 p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <s.icon className="h-4 w-4 text-brand-600" />
                      <span className="text-sm font-semibold text-slate-800 font-mono">{s.name}</span>
                      <Badge color="blue">:{s.port}</Badge>
                    </div>
                    <p className="text-sm text-slate-500">{s.desc}</p>
                  </div>
                ))}
              </div>
            </SubSection>

            <SubSection title="Key concepts">
              <ul className="list-disc list-inside space-y-1 text-sm">
                <li><strong>Tenant</strong> — isolated workspace identified by UUID; all resources scoped to it.</li>
                <li><strong>API Key</strong> — per-tenant bearer token; created in the management UI or via API.</li>
                <li><strong>Booking</strong> — a time-slot reservation with a customer-ref, service-ref, and lifecycle status.</li>
                <li><strong>Module config</strong> — JSON document validated against a JSON Schema and stored per-tenant. Modules include <code>booking</code>, <code>business-hours</code>, <code>queue</code>, <code>notifications</code>, <code>access</code>, and <code>branding</code>.</li>
                <li><strong>X-Tenant-ID header</strong> — the primary authentication signal; all API calls require it.</li>
              </ul>
            </SubSection>
          </Section>

          {/* ═══════════ QUICK START ═══════════ */}
          <Section id="quickstart" icon={Zap} title="Quick Start">
            <p>Get a booking live in under five minutes using Docker Compose.</p>

            <SubSection title="1 — Clone and start">
              {C("qs1", `git clone https://github.com/SoftLaneIT/serviceforge.git
cd serviceforge
docker compose -f deploy/docker/docker-compose.dev.yml up -d`, "bash")}
              <p className="text-sm text-slate-500">All services start with PostgreSQL, Kafka, and migrations applied automatically.</p>
            </SubSection>

            <SubSection title="2 — Create a tenant">
              {C("qs2", `curl -s -X POST http://localhost:8081/v1/tenants \\
  -H "Content-Type: application/json" \\
  -d '{"name":"Acme Corp","slug":"acme","plan":"pro"}' | jq .`, "bash")}
              {C("qs2r", `{
  "id": "01HZ...",
  "name": "Acme Corp",
  "slug": "acme",
  "plan": "pro",
  "status": "active",
  "createdAt": "2026-04-17T10:00:00Z"
}`, "json")}
            </SubSection>

            <SubSection title="3 — Create an API key">
              {C("qs3", `curl -s -X POST http://localhost:8081/v1/api-keys \\
  -H "Content-Type: application/json" \\
  -d '{"name":"acme-prod-key","tenantId":"01HZ..."}' | jq .`, "bash")}
              {C("qs3r", `{
  "id": "key_...",
  "tenantId": "01HZ...",
  "name": "acme-prod-key",
  "key": "sf_live_xxxxxxxxxxxxxxxx",
  "createdAt": "2026-04-17T10:01:00Z"
}`, "json")}
              <Note type="warn">The raw <code>key</code> value is only shown once. Store it securely.</Note>
            </SubSection>

            <SubSection title="4 — Create a booking">
              {C("qs4", `curl -s -X POST http://localhost:8084/v1/bookings \\
  -H "Content-Type: application/json" \\
  -H "X-Tenant-ID: 01HZ..." \\
  -d '{
    "customerRef": "customer_42",
    "serviceRef":  "haircut",
    "slotStart":   "2026-04-20T09:00:00Z",
    "slotEnd":     "2026-04-20T10:00:00Z"
  }' | jq .`, "bash")}
              {C("qs4r", `{
  "id": "bk_...",
  "tenantId": "01HZ...",
  "customerRef": "customer_42",
  "serviceRef":  "haircut",
  "slotStart":   "2026-04-20T09:00:00Z",
  "slotEnd":     "2026-04-20T10:00:00Z",
  "status": "confirmed",
  "metadata": {},
  "createdAt": "2026-04-17T10:02:00Z"
}`, "json")}
            </SubSection>

            <SubSection title="5 — Open the dashboard">
              <p className="text-sm">Navigate to <a href="http://localhost:3000" target="_blank" className="text-brand-600 underline underline-offset-2 hover:text-brand-700">http://localhost:3000</a> to manage tenants, view bookings, configure modules, and set up integrations — no code required.</p>
            </SubSection>
          </Section>

          {/* ═══════════ AUTHENTICATION ═══════════ */}
          <Section id="authentication" icon={Key} title="Authentication">
            <p>Every request to the <strong>booking-service</strong> and <strong>config-service</strong> must carry a tenant identifier via the <code>X-Tenant-ID</code> header. Requests to <strong>management-service</strong> endpoints that operate on behalf of a tenant should also include it.</p>

            <SubSection title="X-Tenant-ID header">
              {C("auth1", `# All booking and config API calls require this header
curl http://localhost:8084/v1/bookings \\
  -H "X-Tenant-ID: <your-tenant-uuid>"`, "bash")}
            </SubSection>

            <SubSection title="Bearer token (API keys)">
              <p className="text-sm">API keys created via the management UI can also be passed as a Bearer token for management-service calls:</p>
              {C("auth2", `curl http://localhost:8081/v1/tenants \\
  -H "Authorization: Bearer sf_live_xxxxxxxxxxxxxxxx"`, "bash")}
              <Note type="info">Both the <code>X-Tenant-ID</code> header and Bearer token can be sent simultaneously when required by a specific endpoint.</Note>
            </SubSection>

            <SubSection title="API key lifecycle">
              <ParamTable rows={[
                { name: "POST /v1/api-keys",        type: "endpoint", required: true,  desc: "Create a new API key for a tenant" },
                { name: "GET  /v1/api-keys",        type: "endpoint", required: false, desc: "List all API keys (optionally filter by tenantId)" },
                { name: "GET  /v1/api-keys/:id",    type: "endpoint", required: false, desc: "Fetch single key metadata (key value not returned)" },
                { name: "DELETE /v1/api-keys/:id",  type: "endpoint", required: false, desc: "Revoke / delete an API key permanently" },
              ]} />
            </SubSection>
          </Section>

          {/* ═══════════ TENANTS API ═══════════ */}
          <Section id="tenants-api" icon={Users} title="Tenants API">
            <p>Base URL: <code className="text-brand-600">http://localhost:8081</code></p>

            <SubSection title="Endpoints">
              <div className="rounded-xl border border-slate-200 overflow-hidden">
                <div className="p-4 space-y-0">
                  <EndpointRow method="GET"    path="/v1/tenants"      desc="List all tenants (paginated)" />
                  <EndpointRow method="POST"   path="/v1/tenants"      desc="Create a new tenant" />
                  <EndpointRow method="GET"    path="/v1/tenants/:id"  desc="Get tenant by ID" />
                  <EndpointRow method="PUT"    path="/v1/tenants/:id"  desc="Update tenant" />
                  <EndpointRow method="DELETE" path="/v1/tenants/:id"  desc="Soft-delete tenant" />
                </div>
              </div>
            </SubSection>

            <SubSection title="Create tenant — request body">
              <ParamTable rows={[
                { name: "name",   type: "string",  required: true,  desc: "Human-readable display name" },
                { name: "slug",   type: "string",  required: true,  desc: "URL-safe identifier, must be unique" },
                { name: "plan",   type: "string",  required: true,  desc: "Plan tier: free | starter | pro | enterprise" },
                { name: "status", type: "string",  required: false, desc: "active | suspended | deleted (default: active)" },
              ]} />
            </SubSection>

            <SubSection title="Example — list tenants">
              {C("tenant1", `curl "http://localhost:8081/v1/tenants?limit=10&offset=0" | jq .`, "bash")}
              {C("tenant1r", `{
  "data": [
    {
      "id": "01HZ...",
      "name": "Acme Corp",
      "slug": "acme",
      "plan": "pro",
      "status": "active",
      "createdAt": "2026-04-17T10:00:00Z"
    }
  ],
  "total": 1
}`, "json")}
            </SubSection>

            <SubSection title="Example — update tenant">
              {C("tenant2", `curl -X PUT http://localhost:8081/v1/tenants/01HZ... \\
  -H "Content-Type: application/json" \\
  -d '{"name":"Acme Corp (Updated)","slug":"acme","plan":"enterprise"}' | jq .`, "bash")}
            </SubSection>
          </Section>

          {/* ═══════════ BOOKINGS API ═══════════ */}
          <Section id="bookings-api" icon={Calendar} title="Bookings API">
            <p>Base URL: <code className="text-brand-600">http://localhost:8084</code> — Requires <code>X-Tenant-ID</code> header on all calls.</p>

            <SubSection title="Endpoints">
              <div className="rounded-xl border border-slate-200 overflow-hidden">
                <div className="p-4 space-y-0">
                  <EndpointRow method="GET"    path="/v1/bookings"          desc="List bookings (filterable by status)" />
                  <EndpointRow method="POST"   path="/v1/bookings"          desc="Create a booking (enforces policy)" />
                  <EndpointRow method="GET"    path="/v1/bookings/:id"      desc="Get booking by ID" />
                  <EndpointRow method="PATCH"  path="/v1/bookings/:id"      desc="Update booking status" />
                  <EndpointRow method="DELETE" path="/v1/bookings/:id"      desc="Cancel a booking" />
                </div>
              </div>
            </SubSection>

            <SubSection title="Create booking — request body">
              <ParamTable rows={[
                { name: "customerRef", type: "string",  required: true,  desc: "Your system's customer identifier" },
                { name: "serviceRef",  type: "string",  required: true,  desc: "Your system's service identifier" },
                { name: "slotStart",   type: "RFC 3339", required: true,  desc: "ISO 8601 UTC datetime, e.g. 2026-04-20T09:00:00Z" },
                { name: "slotEnd",     type: "RFC 3339", required: true,  desc: "Must be after slotStart" },
                { name: "metadata",    type: "object",  required: false, desc: "Arbitrary key-value pairs stored with the booking" },
              ]} />
              <Note type="warn">Always send times in full RFC 3339 format with timezone offset (e.g. <code>2026-04-20T09:00:00Z</code>). The API will reject datetime-local strings like <code>2026-04-20T09:00</code>.</Note>
            </SubSection>

            <SubSection title="Booking statuses">
              <div className="flex flex-wrap gap-2 mt-2">
                {[
                  ["pending",   "blue",   "Created, awaiting confirmation"],
                  ["confirmed", "green",  "Explicitly confirmed or autoConfirm=true"],
                  ["completed", "purple", "Service was delivered"],
                  ["cancelled", "red",    "Cancelled by tenant or customer"],
                  ["no_show",   "yellow", "Customer did not appear"],
                ].map(([s, c, d]) => (
                  <div key={s} className="flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-2 text-sm">
                    <Badge color={c as "blue"}>{s}</Badge>
                    <span className="text-slate-500">{d}</span>
                  </div>
                ))}
              </div>
            </SubSection>

            <SubSection title="Example — list with filter">
              {C("book1", `curl "http://localhost:8084/v1/bookings?status=confirmed&limit=20&offset=0" \\
  -H "X-Tenant-ID: 01HZ..." | jq .`, "bash")}
            </SubSection>

            <SubSection title="Example — update status">
              {C("book2", `curl -X PATCH http://localhost:8084/v1/bookings/bk_... \\
  -H "Content-Type: application/json" \\
  -H "X-Tenant-ID: 01HZ..." \\
  -d '{"status":"completed"}' | jq .`, "bash")}
            </SubSection>

            <SubSection title="Example — cancel">
              {C("book3", `# DELETE returns 204 No Content on success
curl -X DELETE http://localhost:8084/v1/bookings/bk_... \\
  -H "X-Tenant-ID: 01HZ..."`, "bash")}
            </SubSection>
          </Section>

          {/* ═══════════ CONFIG API ═══════════ */}
          <Section id="config-api" icon={Settings} title="Config API">
            <p>Base URL: <code className="text-brand-600">http://localhost:8085</code> — Requires <code>X-Tenant-ID</code> for tenant-scoped operations.</p>

            <SubSection title="Endpoints">
              <div className="rounded-xl border border-slate-200 overflow-hidden">
                <div className="p-4 space-y-0">
                  <EndpointRow method="GET" path="/v1/modules"                    desc="List all available modules with schemas" />
                  <EndpointRow method="GET" path="/v1/modules/:module/schema"     desc="Fetch JSON Schema for a single module" />
                  <EndpointRow method="GET" path="/v1/config/:module"             desc="Read tenant config for a module" />
                  <EndpointRow method="PUT" path="/v1/config/:module"             desc="Upsert tenant config (validated against schema)" />
                  <EndpointRow method="GET" path="/v1/config/:module/history"     desc="Full change history for tenant + module" />
                </div>
              </div>
            </SubSection>

            <SubSection title="Example — read booking config">
              {C("cfg1", `curl http://localhost:8085/v1/config/booking \\
  -H "X-Tenant-ID: 01HZ..." | jq .`, "bash")}
              {C("cfg1r", `{
  "tenantId": "01HZ...",
  "module": "booking",
  "config": {
    "slotDurationMinutes": 60,
    "maxBookingsPerDay": 20,
    "advanceBookingDays": 30,
    "autoConfirm": true,
    "bufferMinutes": 15,
    "cancellationPolicy": {
      "allowCancellation": true,
      "cutoffHours": 24,
      "refundable": true
    }
  },
  "schemaVersion": "1",
  "isDefault": true,
  "updatedAt": "2026-04-17T10:00:00Z"
}`, "json")}
            </SubSection>

            <SubSection title="Example — update booking config">
              {C("cfg2", `curl -X PUT http://localhost:8085/v1/config/booking \\
  -H "Content-Type: application/json" \\
  -H "X-Tenant-ID: 01HZ..." \\
  -d '{
    "slotDurationMinutes": 30,
    "maxBookingsPerDay": 50,
    "advanceBookingDays": 14,
    "autoConfirm": false,
    "bufferMinutes": 10
  }' | jq .`, "bash")}
              <Note type="info">The request body is validated against the module&apos;s JSON Schema. Unrecognised fields are rejected (additionalProperties: false).</Note>
            </SubSection>

            <SubSection title="Example — fetch change history">
              {C("cfg3", `curl "http://localhost:8085/v1/config/booking/history?limit=20" \\
  -H "X-Tenant-ID: 01HZ..." | jq .`, "bash")}
              {C("cfg3r", `{
  "data": [
    {
      "id": "hist_...",
      "tenantId": "01HZ...",
      "module": "booking",
      "config": { "slotDurationMinutes": 30, "..." },
      "changedBy": "system",
      "changedAt": "2026-04-17T11:30:00Z"
    }
  ],
  "total": 1
}`, "json")}
            </SubSection>
          </Section>

          {/* ═══════════ BOOKING POLICY ═══════════ */}
          <Section id="booking-policy" icon={Shield} title="Booking Policy Enforcement">
            <p>When a booking is created, the booking-service reads the tenant&apos;s <code>booking</code> and <code>business-hours</code> module configs (cached for 60 s) and runs a sequential policy pipeline:</p>

            <div className="mt-4 space-y-3">
              {[
                ["1", "Advance booking limit", "blue",   "slotStart must be within now + advanceBookingDays. Prevents bookings too far in the future.", "422"],
                ["2", "Past-slot check",        "blue",   "slotStart must be after the current time. Prevents backdated bookings.", "422"],
                ["3", "Minimum duration",       "blue",   "Slot length must be ≥ slotDurationMinutes. Short bookings are rejected.", "422"],
                ["4", "Business hours",         "green",  "slotStart and slotEnd must fall within the configured day schedule and break window (unless allowBookingsOutsideHours=true).", "422"],
                ["5", "Daily capacity",         "yellow", "CountForDate (excluding cancelled/no_show) must be < maxBookingsPerDay.", "409"],
                ["6", "Buffer gap",             "yellow", "No existing booking for the same serviceRef may fall within ±bufferMinutes of the new slot.", "409"],
                ["7", "Auto-confirm",           "green",  "If autoConfirm=true the booking is created with status=confirmed; otherwise status=pending.", "—"],
              ].map(([n, title, c, desc, code]) => (
                <div key={n} className="flex gap-4 rounded-xl border border-slate-200 p-4">
                  <div className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white ${
                    c === "blue" ? "bg-blue-500" : c === "green" ? "bg-green-500" : "bg-yellow-500"
                  }`}>{n}</div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-semibold text-sm text-slate-800">{title}</span>
                      {code !== "—" && <Badge color={code === "409" ? "yellow" : "red"}>{code}</Badge>}
                    </div>
                    <p className="text-sm text-slate-500">{desc}</p>
                  </div>
                </div>
              ))}
            </div>

            <SubSection title="Graceful degradation">
              <p className="text-sm">If the config-service is unreachable, the booking-service falls back to safe defaults (slotDurationMinutes=30, maxBookingsPerDay=100, advanceBookingDays=365, autoConfirm=false, no business-hours check). Bookings are never rejected due to a config service outage.</p>
            </SubSection>
          </Section>

          {/* ═══════════ MODULES ═══════════ */}
          <Section id="modules" icon={LayoutDashboard} title="Configuration Modules">
            <p>There are six built-in modules. Each is a JSON Schema — defined once, valid forever. Override defaults per-tenant via <code>PUT /v1/config/:module</code>.</p>

            {/* booking */}
            <SubSection title="booking — Core reservation settings">
              <ParamTable rows={[
                { name: "slotDurationMinutes", type: "integer", required: false, desc: "Minimum slot length in minutes (default: 30)" },
                { name: "maxBookingsPerDay",   type: "integer", required: false, desc: "Hard cap on bookings per calendar day (default: 100)" },
                { name: "advanceBookingDays",  type: "integer", required: false, desc: "How far ahead customers can book (default: 365)" },
                { name: "autoConfirm",         type: "boolean", required: false, desc: "Skip 'pending' — go straight to 'confirmed' (default: false)" },
                { name: "bufferMinutes",       type: "integer", required: false, desc: "Gap required between consecutive bookings (default: 0)" },
                { name: "cancellationPolicy",  type: "object",  required: false, desc: "{ allowCancellation, cutoffHours, refundable }" },
              ]} />
            </SubSection>

            {/* business-hours */}
            <SubSection title="business-hours — Operating schedule">
              <ParamTable rows={[
                { name: "timezone",                    type: "string",  required: false, desc: "IANA timezone, e.g. America/New_York (default: UTC)" },
                { name: "allowBookingsOutsideHours",   type: "boolean", required: false, desc: "If true, disable business-hours enforcement (default: false)" },
                { name: "monday … sunday",             type: "object",  required: false, desc: "{ open: boolean, openTime: 'HH:MM', closeTime: 'HH:MM' }" },
                { name: "breakDurationMinutes",        type: "integer", required: false, desc: "Length of midday break (default: 0)" },
                { name: "breakStartTime",              type: "string",  required: false, desc: "Break start in HH:MM, tenant timezone (default: '13:00')" },
              ]} />
              {C("bh1", `{
  "timezone": "America/Chicago",
  "allowBookingsOutsideHours": false,
  "monday":    { "open": true,  "openTime": "08:00", "closeTime": "17:00" },
  "tuesday":   { "open": true,  "openTime": "08:00", "closeTime": "17:00" },
  "saturday":  { "open": false, "openTime": "09:00", "closeTime": "13:00" },
  "sunday":    { "open": false, "openTime": "09:00", "closeTime": "13:00" },
  "breakDurationMinutes": 60,
  "breakStartTime": "12:00"
}`, "json")}
            </SubSection>

            {/* queue */}
            <SubSection title="queue — Waiting list & overflow">
              <ParamTable rows={[
                { name: "lockingTimeoutSeconds",  type: "integer", required: false, desc: "Slot-hold timeout before released (default: 300)" },
                { name: "maxSeats",               type: "integer", required: false, desc: "Maximum concurrent waitlist entries (default: 50)" },
                { name: "queueType",              type: "enum",    required: false, desc: "fifo | priority | fair-share (default: fifo)" },
                { name: "overflowBehaviour",      type: "enum",    required: false, desc: "reject | queue | redirect (default: reject)" },
                { name: "maxWaitTimeSeconds",     type: "integer", required: false, desc: "Drop from queue after this many seconds (default: 1800)" },
                { name: "fairnessPolicy",         type: "object",  required: false, desc: "{ enabled, maxBookingsPerCustomerPerDay }" },
              ]} />
            </SubSection>

            {/* notifications */}
            <SubSection title="notifications — Webhook & email/SMS triggers">
              <ParamTable rows={[
                { name: "webhookEnabled",   type: "boolean", required: false, desc: "Enable outbound webhook delivery (default: false)" },
                { name: "webhookUrl",       type: "string",  required: false, desc: "Target URL for webhook POST requests" },
                { name: "webhookEvents",    type: "array",   required: false, desc: "Events to send: booking.created, booking.confirmed, booking.cancelled, booking.completed, booking.no_show" },
                { name: "emailEnabled",     type: "boolean", required: false, desc: "Enable email notifications (default: false)" },
                { name: "emailTriggers",    type: "object",  required: false, desc: "{ onCreate, onConfirm, onCancel, onReminder }" },
                { name: "smsEnabled",       type: "boolean", required: false, desc: "Enable SMS notifications (default: false)" },
                { name: "smsTriggers",      type: "object",  required: false, desc: "{ onCreate, onConfirm, onCancel }" },
                { name: "retryPolicy",      type: "object",  required: false, desc: "{ maxAttempts, backoffSeconds }" },
              ]} />
            </SubSection>

            {/* access */}
            <SubSection title="access — Session & security controls">
              <ParamTable rows={[
                { name: "sessionTimeoutMinutes",    type: "integer", required: false, desc: "Dashboard session idle timeout (default: 60)" },
                { name: "maxConcurrentSessions",    type: "integer", required: false, desc: "Per-user session cap (default: 5)" },
                { name: "enforceIpAllowlist",       type: "boolean", required: false, desc: "Block requests not in ipAllowlist (default: false)" },
                { name: "ipAllowlist",              type: "array",   required: false, desc: "CIDR or exact IP strings" },
                { name: "requireMfa",               type: "boolean", required: false, desc: "Enforce TOTP/MFA on login (default: false)" },
                { name: "rateLimiting",             type: "object",  required: false, desc: "{ enabled, requestsPerMinute, burstSize }" },
                { name: "loginFailurePolicy",       type: "object",  required: false, desc: "{ maxAttempts, lockoutMinutes }" },
              ]} />
            </SubSection>

            {/* branding */}
            <SubSection title="branding — White-labelling">
              <ParamTable rows={[
                { name: "companyName",   type: "string",  required: false, desc: "Displayed in emails and widget header" },
                { name: "colors",        type: "object",  required: false, desc: "{ primary, accent, background } — hex color values" },
                { name: "logoUrl",       type: "string",  required: false, desc: "HTTPS URL to a PNG/SVG logo" },
                { name: "customDomain",  type: "string",  required: false, desc: "HTTPS domain for widget embedding" },
                { name: "locale",        type: "enum",    required: false, desc: "en | es | fr | de | ja | zh (default: en)" },
                { name: "dateFormat",    type: "string",  required: false, desc: "DD/MM/YYYY, MM/DD/YYYY, YYYY-MM-DD" },
                { name: "timeFormat",    type: "string",  required: false, desc: "12h | 24h (default: 24h)" },
                { name: "hidePoweredBy", type: "boolean", required: false, desc: "Remove 'Powered by ServiceForge' badge" },
              ]} />
              {C("brand1", `curl -X PUT http://localhost:8085/v1/config/branding \\
  -H "Content-Type: application/json" \\
  -H "X-Tenant-ID: 01HZ..." \\
  -d '{
    "companyName": "Acme Corp",
    "colors": {
      "primary":    "#2563eb",
      "accent":     "#16a34a",
      "background": "#f8fafc"
    },
    "logoUrl":      "https://acme.com/logo.png",
    "locale":       "en",
    "timeFormat":   "12h",
    "hidePoweredBy": true
  }' | jq .`, "bash")}
            </SubSection>
          </Section>

          {/* ═══════════ WEBHOOKS ═══════════ */}
          <Section id="webhooks" icon={Webhook} title="Webhooks">
            <p>Enable webhooks in the <code>notifications</code> module. ServiceForge will POST a JSON payload to your <code>webhookUrl</code> whenever a subscribed event fires.</p>

            <SubSection title="Event types">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-2">
                {[
                  ["booking.created",   "New booking submitted"],
                  ["booking.confirmed", "Booking moved to confirmed"],
                  ["booking.cancelled", "Booking was cancelled"],
                  ["booking.completed", "Booking marked completed"],
                  ["booking.no_show",   "Customer no-show recorded"],
                ].map(([e, d]) => (
                  <div key={e} className="flex items-start gap-2 rounded-lg border border-slate-200 p-3">
                    <code className="text-xs font-mono text-brand-600 mt-0.5">{e}</code>
                    <span className="text-sm text-slate-500">{d}</span>
                  </div>
                ))}
              </div>
            </SubSection>

            <SubSection title="Payload shape">
              {C("wh1", `{
  "event":     "booking.confirmed",
  "tenantId":  "01HZ...",
  "timestamp": "2026-04-17T10:05:00Z",
  "data": {
    "id":          "bk_...",
    "customerRef": "customer_42",
    "serviceRef":  "haircut",
    "slotStart":   "2026-04-20T09:00:00Z",
    "slotEnd":     "2026-04-20T10:00:00Z",
    "status":      "confirmed",
    "metadata":    {}
  }
}`, "json")}
            </SubSection>

            <SubSection title="Enable webhooks">
              {C("wh2", `curl -X PUT http://localhost:8085/v1/config/notifications \\
  -H "Content-Type: application/json" \\
  -H "X-Tenant-ID: 01HZ..." \\
  -d '{
    "webhookEnabled": true,
    "webhookUrl":     "https://my-app.example.com/hooks/serviceforge",
    "webhookEvents":  ["booking.created","booking.confirmed","booking.cancelled"],
    "retryPolicy": {
      "maxAttempts":     3,
      "backoffSeconds":  30
    }
  }'`, "bash")}
              <Note type="tip">Your endpoint must respond with HTTP 2xx within 5 seconds or the delivery is retried per the <code>retryPolicy</code>.</Note>
            </SubSection>
          </Section>

          {/* ═══════════ TYPESCRIPT ═══════════ */}
          <Section id="typescript" icon={Code2} title="TypeScript Integration">
            <p>No published npm package yet — copy the thin fetch wrappers from the management-ui source or build your own client using the examples below.</p>

            <SubSection title="Minimal client setup">
              {C("ts1", `// serviceforge.ts
const TENANT_ID = process.env.SERVICEFORGE_TENANT_ID!;
const BOOKING_URL = process.env.SERVICEFORGE_BOOKING_URL ?? "http://localhost:8084";
const CONFIG_URL  = process.env.SERVICEFORGE_CONFIG_URL  ?? "http://localhost:8085";

async function sfRequest<T>(baseUrl: string, path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(\`\${baseUrl}\${path}\`, {
    ...opts,
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-ID": TENANT_ID,
      ...(opts.headers as Record<string, string>),
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const msg = body?.error ?? \`\${res.status} \${res.statusText}\`;
    throw new Error(\`ServiceForge error: \${msg}\`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as T;
}`, "typescript")}
            </SubSection>

            <SubSection title="Create a booking">
              {C("ts2", `interface Booking {
  id: string;
  customerRef: string;
  serviceRef: string;
  slotStart: string;
  slotEnd: string;
  status: "pending" | "confirmed" | "completed" | "cancelled" | "no_show";
  metadata: Record<string, unknown>;
  createdAt: string;
}

async function createBooking(
  customerRef: string,
  serviceRef: string,
  start: Date,
  end: Date,
  metadata?: Record<string, unknown>,
): Promise<Booking> {
  return sfRequest<Booking>(BOOKING_URL, "/v1/bookings", {
    method: "POST",
    body: JSON.stringify({
      customerRef,
      serviceRef,
      slotStart: start.toISOString(), // must be RFC 3339
      slotEnd:   end.toISOString(),
      metadata,
    }),
  });
}

// Usage
const booking = await createBooking(
  "customer_42",
  "haircut",
  new Date("2026-04-20T09:00:00Z"),
  new Date("2026-04-20T10:00:00Z"),
  { notes: "first visit" },
);
console.log(booking.id, booking.status);`, "typescript")}
            </SubSection>

            <SubSection title="Read and update config">
              {C("ts3", `interface BookingConfig {
  slotDurationMinutes: number;
  maxBookingsPerDay: number;
  advanceBookingDays: number;
  autoConfirm: boolean;
  bufferMinutes: number;
}

async function getBookingConfig(): Promise<BookingConfig> {
  const res = await sfRequest<{ config: BookingConfig }>(CONFIG_URL, "/v1/config/booking");
  return res.config;
}

async function setAutoConfirm(enabled: boolean): Promise<void> {
  const current = await getBookingConfig();
  await sfRequest(CONFIG_URL, "/v1/config/booking", {
    method: "PUT",
    body: JSON.stringify({ ...current, autoConfirm: enabled }),
  });
}

await setAutoConfirm(true);`, "typescript")}
            </SubSection>

            <SubSection title="List bookings with pagination">
              {C("ts4", `async function listBookings(
  status?: "pending" | "confirmed" | "completed" | "cancelled" | "no_show",
  limit = 20,
  offset = 0,
) {
  const qs = new URLSearchParams();
  if (status) qs.set("status", status);
  qs.set("limit",  String(limit));
  qs.set("offset", String(offset));
  return sfRequest<{ data: Booking[]; total: number }>(
    BOOKING_URL,
    \`/v1/bookings?\${qs}\`,
  );
}

const page = await listBookings("confirmed", 10, 0);
console.log(\`\${page.total} confirmed bookings\`);
page.data.forEach((b) => console.log(b.id, b.slotStart));`, "typescript")}
            </SubSection>
          </Section>

          {/* ═══════════ ERRORS ═══════════ */}
          <Section id="errors" icon={AlertCircle} title="Error Reference">
            <p>All error responses use a consistent JSON envelope:</p>
            {C("err0", `{ "error": "human-readable message" }`, "json")}

            <SubSection title="HTTP status codes">
              <div className="rounded-xl border border-slate-200 overflow-hidden mt-2">
                <table className="w-full text-sm">
                  <thead className="bg-slate-50">
                    <tr>
                      <th className="px-4 py-2.5 text-left font-semibold text-slate-600 w-20">Code</th>
                      <th className="px-4 py-2.5 text-left font-semibold text-slate-600">Meaning</th>
                      <th className="px-4 py-2.5 text-left font-semibold text-slate-600 hidden sm:table-cell">Common cause</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {[
                      ["200", "OK",                     "Request succeeded, body contains data"],
                      ["201", "Created",                "Resource created (POST)"],
                      ["204", "No Content",             "Success with no body (DELETE)"],
                      ["400", "Bad Request",            "Missing required field or malformed JSON"],
                      ["404", "Not Found",              "Resource does not exist or wrong tenant"],
                      ["409", "Conflict",               "Daily cap hit, buffer gap conflict, or duplicate"],
                      ["422", "Unprocessable Entity",   "Policy violation (past slot, outside hours, too short)"],
                      ["500", "Internal Server Error",  "Unexpected server error — check service logs"],
                    ].map(([code, name, cause]) => (
                      <tr key={code} className="hover:bg-slate-50/50">
                        <td className="px-4 py-2.5"><Badge color={
                          Number(code) < 300 ? "green" :
                          Number(code) < 400 ? "blue" :
                          Number(code) < 500 ? "yellow" : "red"
                        }>{code}</Badge></td>
                        <td className="px-4 py-2.5 font-medium text-slate-800">{name}</td>
                        <td className="px-4 py-2.5 text-slate-500 hidden sm:table-cell">{cause}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </SubSection>

            <SubSection title="Common booking errors">
              {C("err1", `# Slot in the past
{"error": "slot start is in the past"}

# Advance booking limit exceeded
{"error": "slot start exceeds advance booking limit of 30 days"}

# Slot too short
{"error": "slot duration (20 min) is shorter than the minimum configured slot duration (30 min)"}

# Outside business hours
{"error": "slot falls outside business hours for Saturday"}

# Daily cap reached
{"error": "daily booking cap of 20 reached for this date"}

# Buffer conflict
{"error": "booking conflicts with an existing booking within the 15-minute buffer"}`, "text")}
            </SubSection>
          </Section>

          {/* ═══════════ SELF-HOSTING ═══════════ */}
          <Section id="docker" icon={Terminal} title="Self-hosting with Docker">
            <p>The entire platform is containerised. The dev compose file starts everything you need.</p>

            <SubSection title="Start all services">
              {C("dkr1", `# First-time start (builds images, runs DB migrations)
docker compose -f deploy/docker/docker-compose.dev.yml up --build -d

# View logs
docker compose -f deploy/docker/docker-compose.dev.yml logs -f

# Stop everything
docker compose -f deploy/docker/docker-compose.dev.yml down`, "bash")}
            </SubSection>

            <SubSection title="Run DB migrations manually">
              {C("dkr2", `docker compose -f deploy/docker/docker-compose.dev.yml run --rm --no-deps migrate`, "bash")}
            </SubSection>

            <SubSection title="Environment variables — booking-service">
              <ParamTable rows={[
                { name: "PORT",               type: "string",  required: false, desc: "HTTP port (default: 8084)" },
                { name: "DATABASE_URL",        type: "string",  required: true,  desc: "PostgreSQL DSN" },
                { name: "CONFIG_SERVICE_URL",  type: "string",  required: false, desc: "Config service base URL (default: http://localhost:8085)" },
                { name: "CONFIG_CACHE_TTL",    type: "integer", required: false, desc: "Config cache TTL in seconds (default: 60)" },
                { name: "KAFKA_BROKERS",       type: "string",  required: false, desc: "Comma-separated broker addresses (default: localhost:9092)" },
                { name: "KAFKA_ENABLED",       type: "string",  required: false, desc: "Set to false to disable event publishing (default: true)" },
                { name: "CORS_ORIGINS",        type: "string",  required: false, desc: "Comma-separated allowed origins (default: http://localhost:3000)" },
                { name: "LOG_LEVEL",           type: "string",  required: false, desc: "debug | info | warn | error (default: info)" },
              ]} />
            </SubSection>

            <SubSection title="Service ports quick reference">
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-2">
                {[
                  ["management-ui",      "3000"],
                  ["management-service", "8081"],
                  ["booking-service",    "8084"],
                  ["config-service",     "8085"],
                  ["PostgreSQL",         "5432"],
                  ["Kafka",              "9092"],
                  ["Kafka UI",           "8080"],
                ].map(([svc, port]) => (
                  <div key={svc} className="rounded-lg border border-slate-200 p-3 text-center">
                    <div className="text-lg font-bold text-brand-600">:{port}</div>
                    <div className="text-xs text-slate-500 mt-1">{svc}</div>
                  </div>
                ))}
              </div>
            </SubSection>
          </Section>

          {/* footer */}
          <div className="border-t border-slate-200 pt-8 pb-16 text-center text-sm text-slate-400">
            <p>ServiceForge Developer Docs · v0.1.0 · <a href="https://softlaneit.com" target="_blank" className="hover:text-slate-600 underline underline-offset-2">SoftlaneIT</a></p>
            <p className="mt-1">Found an issue? <a href="https://github.com/SoftLaneIT/serviceforge/issues" target="_blank" className="hover:text-slate-600 underline underline-offset-2">Open a GitHub issue</a></p>
          </div>
        </main>
      </div>
    </div>
  );
}
