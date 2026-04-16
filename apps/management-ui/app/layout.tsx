/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 */

import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "@/lib/providers";
import { ToastProvider } from "@/components/ui/toast";

export const metadata: Metadata = {
  title: "ServiceForge Management",
  description: "Multi-tenant service capability management platform",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <Providers>
          <ToastProvider>{children}</ToastProvider>
        </Providers>
      </body>
    </html>
  );
}
