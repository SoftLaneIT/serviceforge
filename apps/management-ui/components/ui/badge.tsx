import { cn } from "@/lib/utils";

export type BadgeVariant =
  | "default"
  | "success"
  | "warning"
  | "error"
  | "info"
  | "neutral";

const variantClasses: Record<BadgeVariant, string> = {
  default:  "bg-brand-50 text-brand-700 border-brand-200",
  success:  "bg-green-50 text-green-700 border-green-200",
  warning:  "bg-amber-50 text-amber-700 border-amber-200",
  error:    "bg-red-50 text-red-700 border-red-200",
  info:     "bg-blue-50 text-blue-700 border-blue-200",
  neutral:  "bg-slate-100 text-slate-600 border-slate-200",
};

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
}

export function Badge({ variant = "default", className, children, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium",
        variantClasses[variant],
        className,
      )}
      {...props}
    >
      {children}
    </span>
  );
}

// Convenience helpers for domain status values.
export function TenantStatusBadge({ status }: { status: string }) {
  const map: Record<string, BadgeVariant> = {
    active:    "success",
    suspended: "warning",
    deleted:   "error",
  };
  return <Badge variant={map[status] ?? "neutral"}>{status}</Badge>;
}

export function KeyStatusBadge({ status }: { status: string }) {
  const map: Record<string, BadgeVariant> = {
    active:  "success",
    revoked: "error",
    expired: "warning",
  };
  return <Badge variant={map[status] ?? "neutral"}>{status}</Badge>;
}

export function BookingStatusBadge({ status }: { status: string }) {
  const map: Record<string, BadgeVariant> = {
    pending:   "warning",
    confirmed: "info",
    completed: "success",
    cancelled: "neutral",
    no_show:   "error",
  };
  return (
    <Badge variant={map[status] ?? "neutral"}>
      {status.replace("_", " ")}
    </Badge>
  );
}

export function EnvBadge({ env }: { env: string }) {
  return (
    <Badge variant={env === "production" ? "default" : "neutral"}>
      {env}
    </Badge>
  );
}

export function PlanBadge({ plan }: { plan: string }) {
  const map: Record<string, BadgeVariant> = {
    starter:    "neutral",
    pro:        "info",
    enterprise: "default",
  };
  return <Badge variant={map[plan] ?? "neutral"}>{plan}</Badge>;
}
