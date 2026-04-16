const BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public body?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  tenantId?: string;
  apiKey?: string;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { body, tenantId, apiKey, headers: extraHeaders, ...rest } = opts;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(extraHeaders as Record<string, string>),
  };

  if (tenantId) headers["X-Tenant-ID"] = tenantId;
  if (apiKey) headers["Authorization"] = `Bearer ${apiKey}`;

  const res = await fetch(`${BASE_URL}${path}`, {
    ...rest,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let errBody: unknown;
    try { errBody = await res.json(); } catch { /* ignore */ }
    const msg =
      errBody && typeof errBody === "object" && "error" in errBody
        ? String((errBody as { error: string }).error)
        : `${res.status} ${res.statusText}`;
    throw new ApiError(res.status, msg, errBody);
  }

  if (res.status === 204) return undefined as T;

  return res.json() as Promise<T>;
}

export const api = {
  get: <T>(path: string, opts?: RequestOptions) =>
    request<T>(path, { method: "GET", ...opts }),
  post: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { method: "POST", body, ...opts }),
  put: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { method: "PUT", body, ...opts }),
  patch: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { method: "PATCH", body, ...opts }),
  delete: <T>(path: string, opts?: RequestOptions) =>
    request<T>(path, { method: "DELETE", ...opts }),
};
