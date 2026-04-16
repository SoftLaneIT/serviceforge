"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import type { Tenant } from "@/lib/api/tenants";

interface TenantContextValue {
  activeTenant: Tenant | null;
  setActiveTenant: (tenant: Tenant | null) => void;
  activeTenantId: string;
}

const TenantContext = createContext<TenantContextValue>({
  activeTenant: null,
  setActiveTenant: () => {},
  activeTenantId: "",
});

const STORAGE_KEY = "sf_active_tenant";

export function TenantProvider({ children }: { children: ReactNode }) {
  const [activeTenant, setActiveTenantState] = useState<Tenant | null>(null);

  // Rehydrate from localStorage on mount
  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) setActiveTenantState(JSON.parse(raw) as Tenant);
    } catch {
      // ignore
    }
  }, []);

  const setActiveTenant = useCallback((tenant: Tenant | null) => {
    setActiveTenantState(tenant);
    if (tenant) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(tenant));
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  }, []);

  return (
    <TenantContext.Provider
      value={{
        activeTenant,
        setActiveTenant,
        activeTenantId: activeTenant?.id ?? "",
      }}
    >
      {children}
    </TenantContext.Provider>
  );
}

export function useTenant() {
  return useContext(TenantContext);
}
