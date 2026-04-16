import { api } from "./client";

export interface ModuleSummary {
  module: string;
  schemaVersion: string;
  description: string;
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
  createdAt: string;
}

export const configApi = {
  listModules(): Promise<ModuleSummary[]> {
    return api.get("/v1/modules");
  },

  getSchema(module: string): Promise<Record<string, unknown>> {
    return api.get(`/v1/modules/${module}/schema`);
  },

  getConfig(module: string, tenantId: string): Promise<ModuleConfig> {
    return api.get(`/v1/config/${module}`, { tenantId });
  },

  upsertConfig(
    module: string,
    config: Record<string, unknown>,
    tenantId: string,
  ): Promise<ModuleConfig> {
    return api.put(`/v1/config/${module}`, { config }, { tenantId });
  },

  getHistory(module: string, tenantId: string): Promise<ConfigHistoryEntry[]> {
    return api.get(`/v1/config/${module}/history`, { tenantId });
  },
};
