/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTenant } from "@/lib/context/tenant-context";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Shell } from "@/components/layout/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Dialog } from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from "@/components/ui/table";
import { BookingStatusBadge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { PageSpinner } from "@/components/ui/spinner";
import { useToast } from "@/components/ui/toast";
import { tenantsApi } from "@/lib/api/tenants";
import {
  bookingsApi,
  type Booking,
  type BookingStatus,
  type CreateBookingBody,
  type EditBookingBody,
} from "@/lib/api/bookings";
import { formatDate } from "@/lib/utils";
import { CalendarDays, Plus, Trash2, RefreshCw, Pencil } from "lucide-react";

const NEXT_STATUSES: Record<BookingStatus, BookingStatus[]> = {
  pending:   ["confirmed", "cancelled"],
  confirmed: ["completed", "cancelled"],
  completed: [],
  cancelled: [],
  no_show:   [],
};

const bookingFormSchema = z.object({
  customerRef: z.string().min(1, "Required"),
  serviceRef:  z.string().min(1, "Required"),
  slotStart:   z.string().min(1, "Required"),
  slotEnd:     z.string().min(1, "Required"),
  metadata:    z.string().optional(),
});
type BookingForm = z.infer<typeof bookingFormSchema>;

const statusFilterOptions = [
  { value: "", label: "All statuses" },
  { value: "pending",   label: "Pending" },
  { value: "confirmed", label: "Confirmed" },
  { value: "completed", label: "Completed" },
  { value: "cancelled", label: "Cancelled" },
  { value: "no_show",   label: "No show" },
];

export default function BookingsPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const [selectedTenantId, setSelectedTenantId] = useState(activeTenant?.id ?? "");

  useEffect(() => {
    if (activeTenant?.id) { setSelectedTenantId(activeTenant.id); setPage(0); }
  }, [activeTenant?.id]);

  const [statusFilter, setStatusFilter] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<Booking | null>(null);
  const [cancelTarget, setCancelTarget] = useState<Booking | null>(null);
  const [updateTarget, setUpdateTarget] = useState<Booking | null>(null);
  const [page, setPage] = useState(0);
  const limit = 20;

  const tenantsQuery = useQuery({
    queryKey: ["tenants", "active-all"],
    queryFn: () => tenantsApi.list({ status: "active", limit: 200 }),
  });

  const tenantOptions = [
    { value: "", label: "Select a tenant…" },
    ...(tenantsQuery.data?.data ?? []).map((t) => ({
      value: t.id,
      label: `${t.name} (${t.slug})`,
    })),
  ];

  const bookingsQueryKey = ["bookings", selectedTenantId, statusFilter, page];
  const bookingsQuery = useQuery({
    queryKey: bookingsQueryKey,
    queryFn: () =>
      bookingsApi.list(selectedTenantId, {
        status: (statusFilter as BookingStatus) || undefined,
        limit,
        offset: page * limit,
      }),
    enabled: !!selectedTenantId,
  });

  // ── Forms ──────────────────────────────────────────────────────────────────
  const createForm = useForm<BookingForm>({ resolver: zodResolver(bookingFormSchema) });
  const { register, handleSubmit, formState: { errors } } = createForm;

  const editForm = useForm<BookingForm>({ resolver: zodResolver(bookingFormSchema) });

  // ── Mutations ──────────────────────────────────────────────────────────────
  const createMutation = useMutation({
    mutationFn: (body: CreateBookingBody) =>
      bookingsApi.create(body, selectedTenantId),
    onSuccess: () => {
      toast("Booking created", "success");
      qc.invalidateQueries({ queryKey: ["bookings", selectedTenantId] });
      setCreateOpen(false);
      createForm.reset();
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const editMutation = useMutation({
    mutationFn: ({ bid, body }: { bid: string; body: EditBookingBody }) =>
      bookingsApi.edit(bid, body, selectedTenantId),
    onSuccess: () => {
      toast("Booking updated", "success");
      qc.invalidateQueries({ queryKey: ["bookings", selectedTenantId] });
      setEditTarget(null);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: BookingStatus }) =>
      bookingsApi.updateStatus(id, { status }, selectedTenantId),
    onSuccess: () => {
      toast("Booking updated", "success");
      qc.invalidateQueries({ queryKey: ["bookings", selectedTenantId] });
      setUpdateTarget(null);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => bookingsApi.cancel(id, selectedTenantId),
    onSuccess: () => {
      toast("Booking cancelled", "success");
      qc.invalidateQueries({ queryKey: ["bookings", selectedTenantId] });
      setCancelTarget(null);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  // ── Edit helpers ───────────────────────────────────────────────────────────
  const openEdit = (b: Booking) => {
    const toLocal = (iso: string) => {
      const d = new Date(iso);
      const pad = (n: number) => String(n).padStart(2, "0");
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
    };
    editForm.reset({
      customerRef: b.customerRef,
      serviceRef:  b.serviceRef,
      slotStart:   toLocal(b.slotStart),
      slotEnd:     toLocal(b.slotEnd),
      metadata:    Object.keys(b.metadata ?? {}).length > 0
        ? JSON.stringify(b.metadata, null, 2)
        : "",
    });
    setEditTarget(b);
  };

  const submitEdit = editForm.handleSubmit((v) => {
    if (!editTarget) return;
    let parsedMeta: Record<string, unknown> = {};
    if (v.metadata?.trim()) {
      try { parsedMeta = JSON.parse(v.metadata); }
      catch { editForm.setError("metadata", { message: "Invalid JSON" }); return; }
    }
    editMutation.mutate({
      bid: editTarget.id,
      body: {
        customerRef: v.customerRef,
        serviceRef:  v.serviceRef,
        slotStart:   new Date(v.slotStart).toISOString(),
        slotEnd:     new Date(v.slotEnd).toISOString(),
        metadata:    parsedMeta,
      },
    });
  });

  const bookings = bookingsQuery.data?.data ?? [];
  const total = bookingsQuery.data?.total ?? 0;
  const totalPages = Math.ceil(total / limit);

  return (
    <Shell>
      <div className="space-y-5 max-w-6xl">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Bookings</h2>
            <p className="text-sm text-slate-500 mt-0.5">
              {selectedTenantId ? `${total} booking${total !== 1 ? "s" : ""}` : "Select a tenant to view bookings"}
            </p>
          </div>
          <Button disabled={!selectedTenantId} onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            New Booking
          </Button>
        </div>

        {/* Filters */}
        <div className="flex gap-3 flex-wrap">
          <div className="w-64">
            <Select
              options={tenantOptions}
              value={selectedTenantId}
              onChange={(e) => { setSelectedTenantId(e.target.value); setPage(0); }}
            />
          </div>
          {selectedTenantId && (
            <div className="w-44">
              <Select
                options={statusFilterOptions}
                value={statusFilter}
                onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}
              />
            </div>
          )}
        </div>

        <Card>
          <CardContent className="p-0">
            {!selectedTenantId ? (
              <div className="py-16 text-center text-sm text-slate-500">
                Select a tenant to view bookings.
              </div>
            ) : bookingsQuery.isLoading ? (
              <PageSpinner />
            ) : bookings.length === 0 ? (
              <EmptyState
                icon={CalendarDays}
                title="No bookings"
                description="No bookings found for the selected filters."
                action={
                  <Button onClick={() => setCreateOpen(true)}>
                    <Plus className="h-4 w-4" />
                    New Booking
                  </Button>
                }
              />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Customer Ref</TableHead>
                    <TableHead>Service Ref</TableHead>
                    <TableHead>Slot Start</TableHead>
                    <TableHead>Slot End</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="w-28" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {bookings.map((b) => {
                    const nextStatuses = NEXT_STATUSES[b.status];
                    return (
                      <TableRow key={b.id}>
                        <TableCell className="font-medium">{b.customerRef}</TableCell>
                        <TableCell>{b.serviceRef}</TableCell>
                        <TableCell className="text-xs">{formatDate(b.slotStart)}</TableCell>
                        <TableCell className="text-xs">{formatDate(b.slotEnd)}</TableCell>
                        <TableCell><BookingStatusBadge status={b.status} /></TableCell>
                        <TableCell className="text-xs text-slate-500">
                          {formatDate(b.createdAt)}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            {(b.status === "pending" || b.status === "confirmed") && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-slate-500 hover:bg-slate-100"
                                title="Edit booking"
                                onClick={() => openEdit(b)}
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </Button>
                            )}
                            {nextStatuses.length > 0 && (
                              <Button
                                variant="ghost"
                                size="sm"
                                title="Update status"
                                onClick={() => setUpdateTarget(b)}
                              >
                                <RefreshCw className="h-3.5 w-3.5" />
                              </Button>
                            )}
                            {(b.status === "pending" || b.status === "confirmed") && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-red-500 hover:bg-red-50"
                                onClick={() => setCancelTarget(b)}
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </Button>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </CardContent>

          {totalPages > 1 && (
            <div className="flex items-center justify-between px-5 py-3 border-t border-slate-100">
              <span className="text-xs text-slate-500">Page {page + 1} of {totalPages}</span>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page + 1 >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </Card>
      </div>

      {/* ── Create booking dialog ────────────────────────────────────────────── */}
      <Dialog
        open={createOpen}
        onClose={() => { setCreateOpen(false); createForm.reset(); }}
        title="New Booking"
        description="Create a booking for this tenant."
      >
        <form
          onSubmit={handleSubmit((v) => {
            let meta: Record<string, unknown> = {};
            if (v.metadata?.trim()) {
              try { meta = JSON.parse(v.metadata); }
              catch { createForm.setError("metadata", { message: "Invalid JSON" }); return; }
            }
            createMutation.mutate({
              customerRef: v.customerRef,
              serviceRef:  v.serviceRef,
              slotStart:   new Date(v.slotStart).toISOString(),
              slotEnd:     new Date(v.slotEnd).toISOString(),
              metadata:    meta,
            });
          })}
          className="space-y-4"
        >
          <Input
            label="Customer Reference"
            placeholder="cust_12345"
            error={errors.customerRef?.message}
            {...register("customerRef")}
          />
          <Input
            label="Service Reference"
            placeholder="svc_abc"
            error={errors.serviceRef?.message}
            {...register("serviceRef")}
          />
          <Input
            label="Slot Start"
            type="datetime-local"
            error={errors.slotStart?.message}
            {...register("slotStart")}
          />
          <Input
            label="Slot End"
            type="datetime-local"
            error={errors.slotEnd?.message}
            {...register("slotEnd")}
          />
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">
              Metadata <span className="text-slate-400 font-normal">(optional JSON)</span>
            </label>
            <textarea
              rows={3}
              placeholder={'{"notes": "first visit"}'}
              className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none"
              {...register("metadata")}
            />
            {errors.metadata && (
              <p className="mt-1 text-xs text-red-500">{errors.metadata.message}</p>
            )}
          </div>
          <div className="flex justify-end gap-2 pt-1">
            <Button type="button" variant="outline" onClick={() => { setCreateOpen(false); createForm.reset(); }}>
              Cancel
            </Button>
            <Button type="submit" loading={createMutation.isPending}>
              Create Booking
            </Button>
          </div>
        </form>
      </Dialog>

      {/* ── Edit booking dialog ──────────────────────────────────────────────── */}
      {editTarget && (
        <Dialog
          open
          onClose={() => setEditTarget(null)}
          title="Edit Booking"
          description={`Editing booking for "${editTarget.customerRef}"`}
        >
          <form onSubmit={submitEdit} className="space-y-4">
            <Input
              label="Customer Reference"
              placeholder="cust_12345"
              error={editForm.formState.errors.customerRef?.message}
              {...editForm.register("customerRef")}
            />
            <Input
              label="Service Reference"
              placeholder="svc_abc"
              error={editForm.formState.errors.serviceRef?.message}
              {...editForm.register("serviceRef")}
            />
            <Input
              label="Slot Start"
              type="datetime-local"
              error={editForm.formState.errors.slotStart?.message}
              {...editForm.register("slotStart")}
            />
            <Input
              label="Slot End"
              type="datetime-local"
              error={editForm.formState.errors.slotEnd?.message}
              {...editForm.register("slotEnd")}
            />
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                Metadata <span className="text-slate-400 font-normal">(optional JSON)</span>
              </label>
              <textarea
                rows={3}
                placeholder={'{"notes": ""}'}
                className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none"
                {...editForm.register("metadata")}
              />
              {editForm.formState.errors.metadata && (
                <p className="mt-1 text-xs text-red-500">{editForm.formState.errors.metadata.message}</p>
              )}
            </div>
            <div className="rounded-lg bg-amber-50 border border-amber-200 px-3 py-2 text-xs text-amber-700">
              Status <strong>{editTarget.status}</strong> is unchanged. Use the status button to change it.
            </div>
            <div className="flex justify-end gap-2 pt-1">
              <Button type="button" variant="outline" onClick={() => setEditTarget(null)}>
                Cancel
              </Button>
              <Button type="submit" loading={editMutation.isPending}>
                Save Changes
              </Button>
            </div>
          </form>
        </Dialog>
      )}

      {/* ── Status update dialog ─────────────────────────────────────────────── */}
      {updateTarget && (
        <Dialog
          open
          onClose={() => setUpdateTarget(null)}
          title="Update Booking Status"
          description={`Current: ${updateTarget.status}`}
          size="sm"
        >
          <div className="space-y-2">
            {NEXT_STATUSES[updateTarget.status].map((s) => (
              <Button
                key={s}
                variant="outline"
                className="w-full justify-start capitalize"
                loading={updateMutation.isPending}
                onClick={() =>
                  updateMutation.mutate({ id: updateTarget.id, status: s })
                }
              >
                Mark as {s.replace("_", " ")}
              </Button>
            ))}
          </div>
        </Dialog>
      )}

      {/* ── Cancel confirm ───────────────────────────────────────────────────── */}
      <ConfirmDialog
        open={!!cancelTarget}
        onClose={() => setCancelTarget(null)}
        onConfirm={() => cancelTarget && cancelMutation.mutate(cancelTarget.id)}
        title="Cancel Booking"
        description={`Cancel booking for "${cancelTarget?.customerRef}"?`}
        confirmLabel="Cancel Booking"
        destructive
        loading={cancelMutation.isPending}
      />
    </Shell>
  );
}
