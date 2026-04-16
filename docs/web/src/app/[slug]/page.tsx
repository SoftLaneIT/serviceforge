import { getPostBySlug, getAllPosts } from "@/lib/api";
import markdownToHtml from "@/lib/markdown";
import { notFound } from "next/navigation";
import { Metadata } from "next";

export async function generateMetadata(props: { params: Promise<{ slug: string }> }): Promise<Metadata> {
  const params = await props.params;
  const post = getPostBySlug(params.slug, ["title", "description"]);
  if (!post.title) {
    return {};
  }
  return {
    title: `${post.title} | ServiceForge Docs`,
    description: post.description,
  };
}

export default async function Post(props: { params: Promise<{ slug: string }> }) {
  const params = await props.params;
  const post = getPostBySlug(params.slug, [
    "title",
    "content",
  ]);

  if (!post.title) {
    return notFound();
  }

  const content = await markdownToHtml(post.content || "");

  return (
    <article className="prose">
      <div dangerouslySetInnerHTML={{ __html: content }} />
    </article>
  );
}

export async function generateStaticParams() {
  const posts = getAllPosts(["slug"]);

  return posts.map((post) => ({
    slug: post.slug,
  }));
}
