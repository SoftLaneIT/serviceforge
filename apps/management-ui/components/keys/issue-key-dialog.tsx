"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Dialog } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useToast } from "@/components/ui/toast";
import { keysApi, type IssueKeyResponse } from "@/lib/api/keys";
import { CheckCircle2, Copy, Check, AlertTriangle } from "lucide-react";

const schema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  environment: z.enum(["sandbox", "production"] as const),
  moduleScope: z.string(),
  expiresAt: z.string().optional(),
});
type FormValues = z.infer<typeof schema>;

const envOptions = [
  { value: "sandbox", label: "Sandbox" },
  { value: "production", label: "Production" },
];

function RawKeyReveal({ rawKey, onClose }: { rawKey: string; onClose: () => void }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(rawKey);
    setCopied(true);
  };
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 rounded-lg bg-amber-50 border border-amber-200 p-3">
        <AlertTriangle className="h-4 w-4 text-amber-500 shrink-0" />
        <p className="text-xs text-amber-800 font-medium">
          Copy this key now — it will not be shown again.
        </p>
      </div>
      <div className="rounded-lg bg-slate-900 p-4 flex items-center gap-3">
        <code className="flex-1 text-xs text-green-400 break-all font-mono">{rawKey}</code>
        <button
          onClick={copy}
          className="shrink-0 text-slate-400 hover:text-white transition-colors"
          title="Copy to clipboard"
        >
          {copied ? (
            <Check className="h-4 w-4 text-green-400" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </button>
      </div>
      <div className="flex justify-end">
        <Button onClick={onClose}>
          <CheckCircle2 className="h-4 w-4" />
          Done — I&apos;ve saved the key
        </Button>
      </div>
    </div>
  );
}

interface IssueKeyDialogProps {
  open: boolean;
  onClose: () => void;
  tenantId: string;
  queryKey: unknown[];
}

export function IssueKeyDialog({ open, onClose, tenantId, queryKey }: IssueKeyDialogProps) {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [issuedKey, setIssuedKey] = useState<IssueKeyResponse | null>(null);

  const issueMutation = useMutation({
    mutationFn: (values: FormValues) =>
      keysApi.issue(
        {
          name: values.name,
          environment: values.environment,
          moduleScope: values.moduleScope
            ? values.moduleScope.split(",").map((s) => s.trim()).filter(Boolean)
            : [],
          expiresAt: values.expiresAt
        ? new Date(values.expiresAt).toISOString()
        : undefined,
        },
        tenantId,
      ),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey });
      setIssuedKey(data);
    },
    onError: (e: Error) => toast(e.message, "error"),
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { environment: "sandbox", moduleScope: "" },
  });

  const handleClose = () => {
    setIssuedKey(null);
    reset();
    onClose();
  };

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      title={issuedKey ? "API Key Issued" : "Issue API Key"}
      description={
        issuedKey
          ? "Your new API key has been created."
          : "Create a new API key for this tenant."
      }
    >
      {issuedKey ? (
        <RawKeyReveal rawKey={issuedKey.rawKey} onClose={handleClose} />
      ) : (
        <form
          onSubmit={handleSubmit((v) => issueMutation.mutate(v))}
          className="space-y-4"
        >
          <Input
            label="Key Name"
            placeholder="mobile-app-prod"
            error={errors.name?.message}
            {...register("name")}
          />
          <Select
            label="Environment"
            options={envOptions}
            error={errors.environment?.message}
            {...register("environment")}
          />
          <Input
            label="Module Scope"
            placeholder="booking, config (leave empty for all)"
            hint="Comma-separated list of modules. Empty = full access."
            error={errors.moduleScope?.message}
            {...register("moduleScope")}
          />
          <Input
            label="Expiry Date"
            type="datetime-local"
            hint="Optional — leave blank for no expiry"
            error={errors.expiresAt?.message}
            {...register("expiresAt")}
          />
          <div className="flex justify-end gap-2 pt-1">
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" loading={issueMutation.isPending}>
              Issue Key
            </Button>
          </div>
        </form>
      )}
    </Dialog>
  );
}
