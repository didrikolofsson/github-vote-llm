import { z } from "zod";

const BASE = "/v1/portal";

async function portalRequest<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: { "Content-Type": "application/json", ...options?.headers },
  });

  const text = await res.text();
  let body: unknown;
  try {
    body = text ? JSON.parse(text) : undefined;
  } catch {
    body = text;
  }

  if (!res.ok) {
    const msg =
      typeof body === "object" && body !== null && "error" in body
        ? String((body as { error: string }).error)
        : `Request failed: ${res.status}`;
    throw new Error(msg);
  }

  return body as T;
}

// ─── Schemas ──────────────────────────────────────────────────────────────────

export const PortalFeatureSchema = z.object({
  id: z.number(),
  title: z.string(),
  description: z.string(),
  review_status: z.enum(["approved", "rejected"]),
  build_status: z
    .enum(["pending", "in_progress", "stuck", "done", "rejected"])
    .nullable(),
  area: z.string().nullable().optional(),
  vote_count: z.number(),
  has_voted: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const PortalPageSchema = z.object({
  org_slug: z.string(),
  repo_owner: z.string(),
  repo_name: z.string(),
  requests: z.array(PortalFeatureSchema),
  pending: z.array(PortalFeatureSchema),
  in_progress: z.array(PortalFeatureSchema),
  done: z.array(PortalFeatureSchema),
});

export const PortalCommentSchema = z.object({
  id: z.number(),
  feature_id: z.number(),
  body: z.string(),
  author_name: z.string(),
  created_at: z.string(),
});

export type PortalFeature = z.infer<typeof PortalFeatureSchema>;
export type PortalPage = z.infer<typeof PortalPageSchema>;
export type PortalComment = z.infer<typeof PortalCommentSchema>;

// ─── API functions ────────────────────────────────────────────────────────────

export async function getPortalPage(
  orgSlug: string,
  repoName: string,
  voterToken: string,
): Promise<PortalPage> {
  const params = voterToken
    ? `?voter_token=${encodeURIComponent(voterToken)}`
    : "";
  const data = await portalRequest<unknown>(`/${orgSlug}/${repoName}${params}`);
  return PortalPageSchema.parse(data);
}

export async function togglePortalVote(
  orgSlug: string,
  repoName: string,
  featureId: number,
  voterToken: string,
  reason: string,
  urgency?: "blocking" | "important" | "nice_to_have",
): Promise<{ vote_count: number }> {
  return portalRequest(`/${orgSlug}/${repoName}/features/${featureId}/vote`, {
    method: "POST",
    body: JSON.stringify({
      voter_token: voterToken,
      reason,
      urgency: urgency ?? "",
    }),
  });
}

export async function listPortalComments(
  orgSlug: string,
  repoName: string,
  featureId: number,
): Promise<PortalComment[]> {
  const data = await portalRequest<{ comments: PortalComment[] }>(
    `/${orgSlug}/${repoName}/features/${featureId}/comments`,
  );
  return z.array(PortalCommentSchema).parse(data.comments);
}

export async function createPortalComment(
  orgSlug: string,
  repoName: string,
  featureId: number,
  body: string,
  authorName?: string,
): Promise<PortalComment> {
  const data = await portalRequest<unknown>(
    `/${orgSlug}/${repoName}/features/${featureId}/comments`,
    {
      method: "POST",
      body: JSON.stringify({ body, author_name: authorName ?? "" }),
    },
  );
  return PortalCommentSchema.parse(data);
}
