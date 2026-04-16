import { ApiError } from "./client";

const CONFIG_BASE =
  process.env.NEXT_PUBLIC_CONFIG_URL ?? "http://localhost:8085";

async function configRequest<T>(
  path: string,
  opts: { method?: string; body?: unknown; tenantId?: string } = {},
): Promise<T> {
  const { method = "GET", body, tenantId } = opts;
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (tenantId) headers["X-Tenant-ID"] = tenantId;

  const res = await fetch(`${CONFIG_BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let errBody: unknown;
    try { errBody = await res.json(); } catch { /* ignore */ }
    throw new ApiError(res.status, `${res.status} ${res.statusText}`, errBody);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// 
// x-ui schema hint types
// 

export interface XUiHint {
  component?: "number_input" | "toggle" | "slider" | "nested_form" | "select" | "text_input" | "color" | "time_input" | "url_input";
  step?: number;
  options?: string[];
}

export interface JsonSchemaProperty {
  type?: string;
  description?: string;
  minimum?: number;
  maximum?: number;
  default?: unknown;
  enum?: unknown[];
  properties?: Record<string, JsonSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
  "x-ui"?: XUiHint;
}

export interface JsonSchema {
  $schema?: string;
  type?: string;
  title?: string;
  description?: string;
  properties?: Record<string, JsonSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
}

// 
// API response types
// 

export interface ModuleSummary {
  module: string;
  version: number;
  schema: JsonSchema;
  defaults: Record<string, unknown>;
  description?: string;
  createdAt: string;
}

export interface ListModulesResponse {
  data: ModuleSummary[];
  total: number;
}

export interface ModuleConfig {
  tenantId: string;
  module: string;
  config: Record<string, unknown>;
  schemaVersion: string;
  isDefault: boolean;
  updatedAt: string;
}

export interface ConfigHistoryEntry {
  id: string;
  tenantId: string;
  module: string;
  config: Record<string, unknown>;
  changedBy: string;
  changedAt: string;
}

export interface ListHistoryResponse {
  data: ConfigHistoryEntry[];
  total: number;
}

// 
// API
// 

export const configApi = {
  listModules(): Promise<ListModulesResponse> {
    return configRequest<ListModulesResponse>("/v1/modules");
  },

  getSchema(module: string): Promise<ModuleSummary> {
    return configRequest<ModuleSummary>(`/v1/modules/${module}/schema`);
  },

  getConfig(module: string, tenantId: string): Promise<ModuleConfig> {
    return configRequest<ModuleConfig>(`/v1/config/${module}`, { tenantId });
  },

  upsertConfig(
    module: string,
    config: Record<string, unknown>,
    tenantId: string,
  ): Promise<ModuleConfig> {
    return configRequest<ModuleConfig>(`/v1/config/${module}`, { method: "PUT", body: config, tenantId });
  },

  getHistory(module: string, tenantId: string): Promise<ListHistoryResponse> {
    return configRequest<ListHistoryResponse>(`/v1/config/${module}/history`, { tenantId });
  },
};
