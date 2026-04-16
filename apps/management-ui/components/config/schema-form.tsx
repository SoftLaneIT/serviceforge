"use client";

import { useState, useEffect, useCallback } from "react";
import { cn } from "@/lib/utils";
import type { JsonSchemaProperty, JsonSchema } from "@/lib/api/config";

// 
// Individual field renderers
// 

function FieldLabel({ name, description, required }: { name: string; description?: string; required?: boolean }) {
  const label = name
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (s) => s.toUpperCase())
    .trim();

  return (
    <div className="mb-1.5">
      <div className="flex items-center gap-1.5">
        <label className="text-sm font-medium text-slate-700">{label}</label>
        {required && <span className="text-xs text-red-500">*</span>}
      </div>
      {description && <p className="text-xs text-slate-400 mt-0.5">{description}</p>}
    </div>
  );
}

// Toggle (boolean)
function ToggleField({
  name,
  value,
  onChange,
  prop,
  required,
}: {
  name: string;
  value: boolean;
  onChange: (v: boolean) => void;
  prop: JsonSchemaProperty;
  required?: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-4 rounded-lg border border-slate-100 bg-white p-3">
      <FieldLabel name={name} description={prop.description} required={required} />
      <button
        type="button"
        role="switch"
        aria-checked={value}
        onClick={() => onChange(!value)}
        className={cn(
          "relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors",
          value ? "bg-brand-600" : "bg-slate-200",
        )}
      >
        <span
          className={cn(
            "pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition-transform",
            value ? "translate-x-4" : "translate-x-0",
          )}
        />
      </button>
    </div>
  );
}

// Number input
function NumberField({
  name,
  value,
  onChange,
  prop,
  required,
}: {
  name: string;
  value: number;
  onChange: (v: number) => void;
  prop: JsonSchemaProperty;
  required?: boolean;
}) {
  return (
    <div className="rounded-lg border border-slate-100 bg-white p-3">
      <FieldLabel name={name} description={prop.description} required={required} />
      <div className="flex items-center gap-2">
        <input
          type="number"
          value={value}
          min={prop.minimum}
          max={prop.maximum}
          onChange={(e) => onChange(Number(e.target.value))}
          className="w-32 rounded-md border border-slate-200 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
        {prop.minimum !== undefined && prop.maximum !== undefined && (
          <span className="text-xs text-slate-400">
            {prop.minimum} – {prop.maximum}
          </span>
        )}
      </div>
    </div>
  );
}

// Slider
function SliderField({
  name,
  value,
  onChange,
  prop,
  required,
}: {
  name: string;
  value: number;
  onChange: (v: number) => void;
  prop: JsonSchemaProperty;
  required?: boolean;
}) {
  const min = prop.minimum ?? 0;
  const max = prop.maximum ?? 100;
  const step = prop["x-ui"]?.step ?? 1;
  const pct = ((value - min) / (max - min)) * 100;

  return (
    <div className="rounded-lg border border-slate-100 bg-white p-3">
      <FieldLabel name={name} description={prop.description} required={required} />
      <div className="flex items-center gap-3 mt-1">
        <div className="relative flex-1 h-2 rounded-full bg-slate-200">
          <div
            className="absolute left-0 top-0 h-2 rounded-full bg-brand-500"
            style={{ width: `${pct}%` }}
          />
          <input
            type="range"
            min={min}
            max={max}
            step={step}
            value={value}
            onChange={(e) => onChange(Number(e.target.value))}
            className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
          />
        </div>
        <span className="w-12 text-right text-sm font-mono font-medium text-slate-800">{value}</span>
        <input
          type="number"
          value={value}
          min={min}
          max={max}
          step={step}
          onChange={(e) => onChange(Number(e.target.value))}
          className="w-20 rounded-md border border-slate-200 px-2 py-1 text-xs focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
      </div>
      <div className="flex justify-between mt-1">
        <span className="text-xs text-slate-400">{min}</span>
        <span className="text-xs text-slate-400">{max}</span>
      </div>
    </div>
  );
}

// String input
function StringField({
  name,
  value,
  onChange,
  prop,
  required,
}: {
  name: string;
  value: string;
  onChange: (v: string) => void;
  prop: JsonSchemaProperty;
  required?: boolean;
}) {
  if (prop.enum && prop.enum.length > 0) {
    return (
      <div className="rounded-lg border border-slate-100 bg-white p-3">
        <FieldLabel name={name} description={prop.description} required={required} />
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full rounded-md border border-slate-200 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
        >
          {prop.enum.map((v) => (
            <option key={String(v)} value={String(v)}>{String(v)}</option>
          ))}
        </select>
      </div>
    );
  }
  return (
    <div className="rounded-lg border border-slate-100 bg-white p-3">
      <FieldLabel name={name} description={prop.description} required={required} />
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-md border border-slate-200 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-500"
      />
    </div>
  );
}

// Nested form section
function NestedSection({
  name,
  value,
  onChange,
  prop,
  required,
}: {
  name: string;
  value: Record<string, unknown>;
  onChange: (v: Record<string, unknown>) => void;
  prop: JsonSchemaProperty;
  required?: boolean;
}) {
  const label = name
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (s) => s.toUpperCase())
    .trim();

  const [open, setOpen] = useState(true);

  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50 overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between px-3 py-2.5 hover:bg-slate-100 transition-colors"
      >
        <div className="text-left">
          <p className="text-sm font-semibold text-slate-700">
            {label}
            {required && <span className="text-xs text-red-500 ml-1">*</span>}
          </p>
          {prop.description && (
            <p className="text-xs text-slate-400">{prop.description}</p>
          )}
        </div>
        <svg
          className={cn("h-4 w-4 text-slate-400 transition-transform", open ? "rotate-180" : "")}
          fill="none" viewBox="0 0 24 24" stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open && prop.properties && (
        <div className="p-3 border-t border-slate-200 space-y-3">
          {Object.entries(prop.properties).map(([key, childProp]) => (
            <SchemaField
              key={key}
              name={key}
              prop={childProp}
              value={value?.[key]}
              required={prop.required?.includes(key)}
              onChange={(v) => onChange({ ...value, [key]: v })}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// 
// Single field dispatcher
// 

function SchemaField({
  name,
  prop,
  value,
  required,
  onChange,
}: {
  name: string;
  prop: JsonSchemaProperty;
  value: unknown;
  required?: boolean;
  onChange: (v: unknown) => void;
}) {
  const ui = prop["x-ui"];
  const component = ui?.component;

  if (prop.type === "boolean" || component === "toggle") {
    return (
      <ToggleField
        name={name}
        prop={prop}
        value={Boolean(value ?? prop.default ?? false)}
        required={required}
        onChange={onChange}
      />
    );
  }

  if (component === "slider" && prop.type === "integer") {
    return (
      <SliderField
        name={name}
        prop={prop}
        value={Number(value ?? prop.default ?? prop.minimum ?? 0)}
        required={required}
        onChange={onChange}
      />
    );
  }

  if (prop.type === "integer" || prop.type === "number" || component === "number_input") {
    return (
      <NumberField
        name={name}
        prop={prop}
        value={Number(value ?? prop.default ?? prop.minimum ?? 0)}
        required={required}
        onChange={onChange}
      />
    );
  }

  if (prop.type === "object" || component === "nested_form") {
    return (
      <NestedSection
        name={name}
        prop={prop}
        value={(value as Record<string, unknown>) ?? (prop.default as Record<string, unknown>) ?? {}}
        required={required}
        onChange={onChange as (v: Record<string, unknown>) => void}
      />
    );
  }

  // string / fallback
  return (
    <StringField
      name={name}
      prop={prop}
      value={String(value ?? prop.default ?? "")}
      required={required}
      onChange={onChange}
    />
  );
}

// 
// Public SchemaForm component
// 

export interface SchemaFormProps {
  schema: JsonSchema;
  defaults: Record<string, unknown>;
  savedConfig: Record<string, unknown>;
  onSave: (config: Record<string, unknown>) => void;
  saving?: boolean;
}

export function SchemaForm({ schema, defaults, savedConfig, onSave, saving }: SchemaFormProps) {
  // Merge: saved values override defaults
  const [values, setValues] = useState<Record<string, unknown>>(() => ({
    ...defaults,
    ...savedConfig,
  }));

  useEffect(() => {
    setValues({ ...defaults, ...savedConfig });
  }, [defaults, savedConfig]);

  const setField = useCallback((key: string, val: unknown) => {
    setValues((prev) => ({ ...prev, [key]: val }));
  }, []);

  const properties = schema.properties ?? {};
  const required = schema.required ?? [];

  // Sort: required fields first, then alphabetical
  const sortedKeys = Object.keys(properties).sort((a, b) => {
    const aReq = required.includes(a) ? 0 : 1;
    const bReq = required.includes(b) ? 0 : 1;
    return aReq - bReq || a.localeCompare(b);
  });

  return (
    <div className="space-y-3">
      {sortedKeys.map((key) => (
        <SchemaField
          key={key}
          name={key}
          prop={properties[key]}
          value={values[key]}
          required={required.includes(key)}
          onChange={(v) => setField(key, v)}
        />
      ))}

      <div className="flex items-center justify-between pt-3 border-t border-slate-100">
        <button
          type="button"
          className="text-xs text-slate-500 hover:text-slate-700 transition-colors"
          onClick={() => setValues({ ...defaults, ...savedConfig })}
        >
          Reset to saved
        </button>
        <button
          type="button"
          disabled={saving}
          onClick={() => onSave(values)}
          className={cn(
            "flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors",
            saving
              ? "bg-brand-400 cursor-not-allowed"
              : "bg-brand-600 hover:bg-brand-700",
          )}
        >
          {saving ? (
            <>
              <svg className="h-3.5 w-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
              Saving…
            </>
          ) : (
            <>
              <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Save Configuration
            </>
          )}
        </button>
      </div>
    </div>
  );
}
