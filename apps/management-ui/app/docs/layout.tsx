import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "ServiceForge Developer Docs",
  description: "Full API reference, integration guides, and developer examples for ServiceForge",
};

export default function DocsLayout({ children }: { children: React.ReactNode }) {
  return children;
}
