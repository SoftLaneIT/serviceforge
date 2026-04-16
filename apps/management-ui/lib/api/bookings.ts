import { api } from "./client";

export type BookingStatus =
  | "pending"
  | "confirmed"
  | "completed"
  | "cancelled"
  | "no_show";

export interface Booking {
  id: string;
  tenantId: string;
  customerRef: string;
  serviceRef: string;
  slotStart: string;
  slotEnd: string;
  status: BookingStatus;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface ListBookingsParams {
  status?: BookingStatus;
  limit?: number;
  offset?: number;
}

export interface ListBookingsResponse {
  bookings: Booking[];
  total: number;
}

export interface CreateBookingBody {
  customerRef: string;
  serviceRef: string;
  slotStart: string;
  slotEnd: string;
  metadata?: Record<string, unknown>;
}

export interface UpdateBookingBody {
  status: BookingStatus;
}

export const bookingsApi = {
  list(tenantId: string, params: ListBookingsParams = {}): Promise<ListBookingsResponse> {
    const qs = new URLSearchParams();
    if (params.status) qs.set("status", params.status);
    if (params.limit !== undefined) qs.set("limit", String(params.limit));
    if (params.offset !== undefined) qs.set("offset", String(params.offset));
    const query = qs.toString() ? `?${qs}` : "";
    return api.get(`/v1/bookings${query}`, { tenantId });
  },

  get(id: string, tenantId: string): Promise<Booking> {
    return api.get(`/v1/bookings/${id}`, { tenantId });
  },

  create(body: CreateBookingBody, tenantId: string): Promise<Booking> {
    return api.post("/v1/bookings", body, { tenantId });
  },

  updateStatus(id: string, body: UpdateBookingBody, tenantId: string): Promise<Booking> {
    return api.patch(`/v1/bookings/${id}`, body, { tenantId });
  },

  cancel(id: string, tenantId: string): Promise<void> {
    return api.delete(`/v1/bookings/${id}`, { tenantId });
  },
};
