import { ApiError } from "./client";

const BOOKING_BASE =
  process.env.NEXT_PUBLIC_BOOKING_URL ?? "http://localhost:8084";

async function bookingRequest<T>(
  path: string,
  opts: { method?: string; body?: unknown; tenantId?: string } = {},
): Promise<T> {
  const { method = "GET", body, tenantId } = opts;
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (tenantId) headers["X-Tenant-ID"] = tenantId;

  const res = await fetch(`${BOOKING_BASE}${path}`, {
    method,
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
  data: Booking[];
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
    return bookingRequest<ListBookingsResponse>(`/v1/bookings${query}`, { tenantId });
  },

  get(id: string, tenantId: string): Promise<Booking> {
    return bookingRequest<Booking>(`/v1/bookings/${id}`, { tenantId });
  },

  create(body: CreateBookingBody, tenantId: string): Promise<Booking> {
    return bookingRequest<Booking>("/v1/bookings", { method: "POST", body, tenantId });
  },

  updateStatus(id: string, body: UpdateBookingBody, tenantId: string): Promise<Booking> {
    return bookingRequest<Booking>(`/v1/bookings/${id}`, { method: "PATCH", body, tenantId });
  },

  cancel(id: string, tenantId: string): Promise<void> {
    return bookingRequest<void>(`/v1/bookings/${id}`, { method: "DELETE", tenantId });
  },
};
