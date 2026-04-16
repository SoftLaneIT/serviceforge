/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

"use client";

import { use, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import Link from "next/link";
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
} from "@/lib/api/bookings";
import { formatDate } from "@/lib/utils";
import { CalendarDays, Plus, Trash2, RefreshCw, ArrowLeft } from "lucide-react";

const NEXT_STATUSES: Record<BookingStatus, BookingStatus[]> = {
  pending:   ["confirmed", "cancelled"],
  confirmed: ["completed", "cancelled"],
  completed: [],
  cancelled: [],
  no_show:   [],
};

const createSchema = z.object({
  customerRef: z.string().min(1, "Required"),
  serviceRef:  z.string().min(1, "Required"),
  slotStart:   z.string().min(1, "Required"),
  slotEnd:     z.string().min(1, "Required"),
});
type CreateForm = z.infer<typeof createSchema>;

const statusFilterOptions = [
  { value: "", label: "All statuses" },
  { value: "pending",   label: "Pending" },
  { value: "confirmed", label: "Confirmed" },
  { value: "completed", label: "Completed" },
  { value: "cancelled", label: "Cancelled" },
  { value: "no_show",   label: "No show" },
];

export default function TenantBookingsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { toast } = useToast();
  const qc = useQueryClient();
  const [statusFilter, setStatusFilter] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [cancelTarget, setCancelTarget] = useState<Booking | null>(null);
  const [updateTarget, setUpdateTarget] = useState<Booking | null>(null);
  const [page, setPage] = useState(0);
  const limit = 20;

  const tenantQuery = useQuery({
    queryKey: ["tenant", id],
    queryFn: () => tenantsApi.get(id),
  });

  const queryKey = ["bookings", id, statusFilter, page];
  const bookingsQuery = useQuery({
    queryKey,
    queryFn: () =>
      bookingsApi.list(id, {
        status: (statusFilter as BookingStatus) || undefined,
        limit,
        offset: page * limit,
      }),
  });

  const createMutation = useMutation({
    mutationFn: (body: CreateBookingBody) => bookingsApi.create(body, id),
    onSuccess: () => {
      toast("Booking created", "success");
      qc.invalidateQueries({ queryKey: ["bookings", id] });
      setCreateOpen(false);
      reset();
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const updateMutation = useMutation({
    mutationFn: ({ bid, status }: { bid: string; status: BookingStatus }) =>
      bookingsApi.updateStatus(bid, { status }, id),
    onSuccess: () => {
      toast("Status updated", "success");
      qc.invalidateQueries({ queryKey: ["bookings", id] });
      setUpdateTarget(null);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const cancelMutation = useMutation({
    mutationFn: (bid: string) => bookingsApi.cancel(bid, id),
    onSuccess: () => {
      toast("Booking cancelled", "success");
      qc.invalidateQueries({ queryKey: ["bookings", id] });
      setCancelTarget(null);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateForm>({ resolver: zodResolver(createSchema) });

  const bookings = bookingsQuery.data?.data ?? [];
  const total = bookingsQuery.data?.total ?? 0;
  const totalPages = Math.ceil(total / limit);

  return (
    <Shell>
      <div className="space-y-5 max-w-6xl">
        <div className="flex items-center gap-3">
          <Link href={`/tenants/${id}`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4" />
              {tenantQuery.data?.name ?? "Tenant"}
            </Button>
          </Link>
        </div>

        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Bookings</h2>
            <p className="text-sm text-slate-500 mt-0.5">
              {total} booking{total !== 1 ? "s" : ""} for{" "}
              <span className="font-medium">{tenantQuery.data?.name ?? id}</span>
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Select
              options={statusFilterOptions}
              value={statusFilter}
              onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}
              className="w-40"
            />
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              New Booking
            </Button>
          </div>
        </div>

        <Card>
          <CardContent className="p-0">
            {bookingsQuery.isLoading ? (
              <PageSpinner />
            ) : bookings.length === 0 ? (
              <EmptyState
                icon={CalendarDays}
                title="No bookings"
                description="No bookings found."
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
                    <TableHead className="w-20" />
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
                            {nextStatuses.length > 0 && (
                              <Button
                                variant="ghost"
                                size="sm"
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
                <Button variant="outline" size="sm" disabled={page + 1 >= totalPages} onClick={() => setPage((p) => p + 1)}>
                  Next
                </Button>
              </div>
            </div>
          )}
        </Card>
      </div>

      <Dialog
        open={createOpen}
        onClose={() => { setCreateOpen(false); reset(); }}
        title="New Booking"
        description="Create a booking for this tenant."
      >
        <form onSubmit={handleSubmit((v) => createMutation.mutate(v))} className="space-y-4">
          <Input label="Customer Reference" placeholder="cust_12345" error={errors.customerRef?.message} {...register("customerRef")} />
          <Input label="Service Reference" placeholder="svc_abc" error={errors.serviceRef?.message} {...register("serviceRef")} />
          <Input label="Slot Start" type="datetime-local" error={errors.slotStart?.message} {...register("slotStart")} />
          <Input label="Slot End" type="datetime-local" error={errors.slotEnd?.message} {...register("slotEnd")} />
          <div className="flex justify-end gap-2 pt-1">
            <Button type="button" variant="outline" onClick={() => { setCreateOpen(false); reset(); }}>Cancel</Button>
            <Button type="submit" loading={createMutation.isPending}>Create Booking</Button>
          </div>
        </form>
      </Dialog>

      {updateTarget && (
        <Dialog open onClose={() => setUpdateTarget(null)} title="Update Status" size="sm">
          <div className="space-y-2">
            {NEXT_STATUSES[updateTarget.status].map((s) => (
              <Button
                key={s}
                variant="outline"
                className="w-full justify-start capitalize"
                loading={updateMutation.isPending}
                onClick={() => updateMutation.mutate({ bid: updateTarget.id, status: s })}
              >
                Mark as {s.replace("_", " ")}
              </Button>
            ))}
          </div>
        </Dialog>
      )}

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
