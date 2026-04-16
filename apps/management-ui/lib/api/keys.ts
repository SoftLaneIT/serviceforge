import { api } from "./client";

export type KeyEnvironment = "sandbox" | "production";
export type KeyStatus = "active" | "revoked";

export interface ApiKey {
  id: string;
  tenantId: string;
  name: string;
  keyPrefix: string;
  environment: KeyEnvironment;
  moduleScope: string[];
  status: KeyStatus;
  lastUsedAt: string | null;
  expiresAt: string | null;
  revokedAt: string | null;
  createdAt: string;
}

export interface IssueKeyBody {
  name: string;
  environment: KeyEnvironment;
  moduleScope?: string[];
  expiresAt?: string;
}

export interface IssueKeyResponse extends ApiKey {
  rawKey: string;
}

export const keysApi = {
  list(tenantId: string): Promise<ApiKey[]> {
    return api.get("/v1/keys", { tenantId });
  },

  get(id: string, tenantId: string): Promise<ApiKey> {
    return api.get(`/v1/keys/${id}`, { tenantId });
  },

  issue(body: IssueKeyBody, tenantId: string): Promise<IssueKeyResponse> {
    return api.post("/v1/keys", body, { tenantId });
  },

  revoke(id: string, tenantId: string): Promise<void> {
    return api.delete(`/v1/keys/${id}`, { tenantId });
  },
};
