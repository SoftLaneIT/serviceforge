import type { Metadata } from "next";
import "./globals.css";
import { getAllPosts } from "@/lib/api";
import Link from "next/link";

export const metadata: Metadata = {
  title: "ServiceForge Documentation",
  description: "A-Z guide for ServiceForge Platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const posts = getAllPosts(['title', 'slug', 'order']);

  return (
    <html lang="en">
      <body>
        <div className="layout-container">
          <aside className="sidebar">
            <div className="sidebar-title">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg>
              ServiceForge
            </div>
            <ul className="nav-list">
              {posts.map((post) => (
                <li key={post.slug} className="nav-item">
                  <Link href={`/${post.slug}`} className="nav-link">
                    <span>{post.title}</span>
                  </Link>
                </li>
              ))}
            </ul>
          </aside>
          <main className="main-content">
            {children}
          </main>
        </div>
      </body>
    </html>
  );
}
