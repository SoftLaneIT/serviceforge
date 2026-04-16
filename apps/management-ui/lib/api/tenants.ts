import { api } from "./client";

export type TenantPlan = "starter" | "pro" | "enterprise";
export type TenantStatus = "active" | "suspended" | "deleted";

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  plan: TenantPlan;
  status: TenantStatus;
  settings: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface ListTenantsParams {
  status?: TenantStatus;
  limit?: number;
  offset?: number;
}

export interface ListTenantsResponse {
  data: Tenant[];
  total: number;
  limit: number;
  offset: number;
}

export interface CreateTenantBody {
  name: string;
  slug: string;
  plan: TenantPlan;
  settings?: Record<string, unknown>;
}

export interface UpdateTenantBody {
  name?: string;
  plan?: TenantPlan;
  settings?: Record<string, unknown>;
}

export const tenantsApi = {
  list(params: ListTenantsParams = {}): Promise<ListTenantsResponse> {
    const qs = new URLSearchParams();
    if (params.status) qs.set("status", params.status);
    if (params.limit !== undefined) qs.set("limit", String(params.limit));
    if (params.offset !== undefined) qs.set("offset", String(params.offset));
    const query = qs.toString() ? `?${qs}` : "";
    return api.get(`/v1/tenants${query}`);
  },

  get(id: string): Promise<Tenant> {
    return api.get(`/v1/tenants/${id}`);
  },

  create(body: CreateTenantBody): Promise<Tenant> {
    return api.post("/v1/tenants", body);
  },

  update(id: string, body: UpdateTenantBody): Promise<Tenant> {
    return api.patch(`/v1/tenants/${id}`, body);
  },

  delete(id: string): Promise<void> {
    return api.delete(`/v1/tenants/${id}`);
  },
};
