// Health checks hit each service directly (bypasses gateway auth).
const SERVICES: Record<string, string> = {
  "api-gateway":    process.env.NEXT_PUBLIC_GATEWAY_URL    ?? "http://localhost:8081",
  "auth-service":   process.env.NEXT_PUBLIC_AUTH_URL       ?? "http://localhost:8082",
  "tenant-service": process.env.NEXT_PUBLIC_TENANT_URL     ?? "http://localhost:8083",
  "booking-service":process.env.NEXT_PUBLIC_BOOKING_URL    ?? "http://localhost:8084",
  "config-service": process.env.NEXT_PUBLIC_CONFIG_URL     ?? "http://localhost:8085",
};

export interface ServiceHealth {
  name: string;
  status: "ok" | "error" | "loading";
  latencyMs?: number;
}

export async function checkHealth(name: string): Promise<ServiceHealth> {
  const base = SERVICES[name];
  const start = Date.now();
  try {
    const res = await fetch(`${base}/health`, {
      signal: AbortSignal.timeout(3000),
      cache: "no-store",
    });
    const latencyMs = Date.now() - start;
    return { name, status: res.ok ? "ok" : "error", latencyMs };
  } catch {
    return { name, status: "error", latencyMs: Date.now() - start };
  }
}

export async function checkAllHealth(): Promise<ServiceHealth[]> {
  return Promise.all(Object.keys(SERVICES).map(checkHealth));
}
