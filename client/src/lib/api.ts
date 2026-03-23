/**
 * API client for vote-llm backend.
 * Uses fetch with Bearer token for protected routes.
 * Validates responses with Zod schemas.
 */

import { z } from "zod";
import { createLogger } from "./logger";

const BASE = "/v1";
const logger = createLogger("api");

let accessToken: string | null = null;

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export function getAccessToken(): string | null {
  return accessToken;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public body?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** Format API/validation errors for display to users. */
export function formatApiError(err: unknown): string {
  if (err instanceof ApiError) {
    if (
      typeof err.body === "object" &&
      err.body !== null &&
      "error" in err.body
    ) {
      return String((err.body as { error: string }).error);
    }
    return err.message;
  }
  if (err instanceof z.ZodError) {
    const parts = err.issues.map(
      (i) => `${i.path.filter(Boolean).join(".") || "response"}: ${i.message}`,
    );
    return `Validation failed: ${parts.join("; ")}`;
  }
  return err instanceof Error ? err.message : String(err);
}

async function request<S extends z.ZodSchema>(
  path: string,
  options: RequestInit & { schema: S; skipAuth?: boolean },
): Promise<z.infer<S>> {
  const { schema, skipAuth, ...init } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init.headers as Record<string, string>),
  };

  if (!skipAuth && accessToken) {
    headers["Authorization"] = `Bearer ${accessToken}`;
  }

  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers,
  });

  const text = await res.text();
  let data: unknown;
  try {
    data = text ? JSON.parse(text) : undefined;
  } catch {
    data = text;
  }

  if (!res.ok) {
    const errBody =
      typeof data === "object" && data !== null && "error" in data
        ? (data as { error: string }).error
        : data;
    throw new ApiError(
      typeof errBody === "string" ? errBody : `Request failed: ${res.status}`,
      res.status,
      data,
    );
  }

  return schema.parseAsync(data).catch((err) => {
    logger.error("Response validation failed", { path, err });
    throw err;
  });
}

// ─── Auth API (no token required) ─────────────────────────────────────────────

import {
  AuthorizeResponseSchema,
  SignupResponseSchema,
  TokenResponseSchema,
} from "./auth-schemas";

export async function authorize(params: {
  email: string;
  password: string;
  code_challenge: string;
  redirect_uri: string;
}) {
  return request("/auth/authorize", {
    method: "POST",
    body: JSON.stringify(params),
    schema: AuthorizeResponseSchema,
    skipAuth: true,
  });
}

export async function exchangeToken(params: {
  grant_type: "authorization_code";
  code: string;
  code_verifier: string;
  redirect_uri: string;
}) {
  return request("/auth/token", {
    method: "POST",
    body: JSON.stringify(params),
    schema: TokenResponseSchema,
    skipAuth: true,
  });
}

export async function refreshToken(refresh_token: string) {
  return request("/auth/token", {
    method: "POST",
    body: JSON.stringify({ grant_type: "refresh_token", refresh_token }),
    schema: TokenResponseSchema,
    skipAuth: true,
  });
}

export async function revokeToken(refresh_token: string) {
  return request("/auth/revoke", {
    method: "POST",
    body: JSON.stringify({ refresh_token }),
    schema: z.void(),
    skipAuth: true,
  });
}

export async function signup(params: { email: string; password: string }) {
  return request("/users/signup", {
    method: "POST",
    body: JSON.stringify(params),
    schema: SignupResponseSchema,
    skipAuth: true,
  });
}

// ─── App API (protected, requires token) ─────────────────────────────────────

import {
  OrganizationListResponseSchema,
  OrganizationWithMembersSchema,
  ProposalCommentListSchema,
  ProposalCommentSchema,
  ProposalListSchema,
  ProposalSchema,
  RepoConfigListSchema,
  RepoConfigSchema,
  RunListSchema,
  RunSchema,
  UpdateRepoConfigRequestSchema,
} from "./api-schemas";

import type {
  Organization,
  Proposal,
  ProposalComment,
  RepoConfig,
  Run,
} from "./api-schemas";
export type {
  Organization,
  OrganizationWithMembers,
  Proposal,
  ProposalComment,
  RepoConfig,
  Run,
  UpdateRepoConfigRequest,
} from "./api-schemas";

let onRefresh: (() => Promise<string | null>) | null = null;

export function setOnRefresh(fn: (() => Promise<string | null>) | null): void {
  onRefresh = fn;
}

async function requestWithRefresh<S extends z.ZodSchema>(
  path: string,
  options: RequestInit & { schema: S; skipAuth?: boolean },
): Promise<z.infer<S>> {
  try {
    return await request(path, options);
  } catch (err) {
    if (
      err instanceof ApiError &&
      err.status === 401 &&
      onRefresh &&
      !options.skipAuth
    ) {
      const newToken = await onRefresh();
      if (newToken) {
        setAccessToken(newToken);
        return request(path, options);
      }
    }
    throw err;
  }
}

export async function listMyOrganizations(): Promise<Organization[]> {
  const data = await requestWithRefresh("/organizations", {
    schema: OrganizationListResponseSchema,
  });
  return data.organizations;
}

export async function createOrganization(name: string) {
  return requestWithRefresh("/organizations", {
    method: "POST",
    body: JSON.stringify({ name }),
    schema: OrganizationWithMembersSchema,
  });
}

// GitHub connection
export const GitHubStatusSchema = z.object({
  connected: z.boolean(),
  login: z.string().nullable().optional(),
});
export async function getGitHubStatus(): Promise<
  z.infer<typeof GitHubStatusSchema>
> {
  return requestWithRefresh("/github/status", {
    schema: GitHubStatusSchema,
  });
}

export const GitHubAuthorizeUrlSchema = z.object({
  authorize_url: z.string(),
});
export async function getGitHubAuthorizeUrl(): Promise<
  z.infer<typeof GitHubAuthorizeUrlSchema>
> {
  return requestWithRefresh("/github/authorize", {
    schema: GitHubAuthorizeUrlSchema,
  });
}

// Organization repositories
export type OrgRepository = {
  owner: string;
  repo: string;
  created_at?: string;
};

const OrgRepositoryListResponseSchema = z.object({
  repositories: z.array(
    z.object({
      owner: z.string(),
      repo: z.string(),
      created_at: z.string().optional(),
    }),
  ),
});

export async function listOrgRepositories(
  orgId: number,
): Promise<OrgRepository[]> {
  const data = await requestWithRefresh(
    `/organizations/${orgId}/repositories`,
    {
      method: "GET",
      schema: OrgRepositoryListResponseSchema,
    },
  );
  return data.repositories;
}

export const AvailableRepositoriesSchema = z.object({
  repositories: z.array(z.object({ owner: z.string(), repo: z.string() })),
  has_more: z.boolean(),
});
export async function listAvailableRepositories(
  orgId: number,
  page = 1,
): Promise<z.infer<typeof AvailableRepositoriesSchema>> {
  return requestWithRefresh(
    `/organizations/${orgId}/repositories/available?page=${page}`,
    {
      schema: AvailableRepositoriesSchema,
    },
  );
}

export async function addRepository(
  orgId: number,
  owner: string,
  repo: string,
): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/repositories`, {
    method: "POST",
    body: JSON.stringify({ owner, repo }),
    schema: z.void(),
  });
}

export async function removeRepository(
  orgId: number,
  owner: string,
  repo: string,
): Promise<void> {
  await requestWithRefresh(
    `/organizations/${orgId}/repositories/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
    { method: "DELETE", schema: z.void() },
  );
}

// Organization members
export const OrgMemberSchema = z.object({
  user_id: z.number(),
  email: z.string(),
  role: z.string(),
});
export type OrgMember = z.infer<typeof OrgMemberSchema>;
export async function listOrgMembers(orgId: number): Promise<OrgMember[]> {
  const data = await requestWithRefresh(`/organizations/${orgId}/members`, {
    schema: z.object({ members: z.array(OrgMemberSchema) }),
  });
  return data.members;
}

export async function inviteMember(
  orgId: number,
  email: string,
): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/members`, {
    method: "POST",
    body: JSON.stringify({ email }),
    schema: z.void(),
  });
}

export async function removeMember(
  orgId: number,
  userId: number,
): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/members/${userId}`, {
    method: "DELETE",
    schema: z.void(),
  });
}

export async function updateMemberRole(
  orgId: number,
  userId: number,
  role: "owner" | "member",
): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/members/${userId}`, {
    method: "PATCH",
    body: JSON.stringify({ role }),
    schema: z.void(),
  });
}

export async function listRepos(): Promise<RepoConfig[]> {
  const data = await requestWithRefresh("/repos", {
    schema: RepoConfigListSchema,
  });
  return data;
}

export async function getRepoConfig(owner: string, repo: string) {
  return requestWithRefresh(`/repos/${owner}/${repo}/config`, {
    schema: RepoConfigSchema,
  });
}

export async function updateRepoConfig(
  owner: string,
  repo: string,
  body: z.infer<typeof UpdateRepoConfigRequestSchema>,
) {
  return requestWithRefresh(`/repos/${owner}/${repo}/config`, {
    method: "PUT",
    body: JSON.stringify(body),
    schema: RepoConfigSchema,
  });
}

export async function deleteRepoConfig(owner: string, repo: string) {
  await requestWithRefresh(`/repos/${owner}/${repo}/config`, {
    method: "DELETE",
    schema: z.void(),
  });
}

export async function listRuns(): Promise<Run[]> {
  const data = await requestWithRefresh("/runs", {
    schema: RunListSchema,
  });
  return data;
}

export async function getRun(id: number) {
  return requestWithRefresh(`/runs/${id}`, {
    schema: RunSchema,
  });
}

export async function cancelRun(id: number): Promise<void> {
  await requestWithRefresh(`/runs/${id}/cancel`, {
    method: "POST",
    schema: z.void(),
  });
}

export async function retryRun(id: number): Promise<void> {
  await requestWithRefresh(`/runs/${id}/retry`, {
    method: "POST",
    schema: z.void(),
  });
}

export async function listRoadmapItems(
  owner: string,
  repo: string,
): Promise<Proposal[]> {
  const data = await requestWithRefresh(`/repos/${owner}/${repo}/roadmap`, {
    schema: ProposalListSchema,
  });
  return data;
}

export async function updateProposalStatus(
  owner: string,
  repo: string,
  id: number,
  body: { status: Proposal["status"] },
) {
  return requestWithRefresh(`/repos/${owner}/${repo}/proposals/${id}`, {
    method: "PATCH",
    body: JSON.stringify(body),
    schema: ProposalSchema,
  });
}

export async function listBoardProposals(
  owner: string,
  repo: string,
): Promise<Proposal[]> {
  const data = await requestWithRefresh(`/board/${owner}/${repo}/proposals`, {
    schema: ProposalListSchema,
  });
  return data;
}

export async function createBoardProposal(
  owner: string,
  repo: string,
  body: { title: string; description?: string; author_name?: string },
) {
  return requestWithRefresh(`/board/${owner}/${repo}/proposals`, {
    method: "POST",
    body: JSON.stringify(body),
    schema: ProposalSchema,
  });
}

export async function voteProposal(
  owner: string,
  repo: string,
  id: number,
): Promise<Proposal> {
  return requestWithRefresh(`/board/${owner}/${repo}/proposals/${id}/vote`, {
    method: "POST",
    schema: ProposalSchema,
  });
}

export async function listBoardComments(
  owner: string,
  repo: string,
  id: number,
): Promise<ProposalComment[]> {
  const data = await requestWithRefresh(
    `/board/${owner}/${repo}/proposals/${id}/comments`,
    {
      schema: ProposalCommentListSchema,
    },
  );
  return data;
}

export async function createBoardComment(
  owner: string,
  repo: string,
  id: number,
  body: { body: string; author_name?: string },
) {
  return requestWithRefresh(
    `/board/${owner}/${repo}/proposals/${id}/comments`,
    {
      method: "POST",
      body: JSON.stringify(body),
      schema: ProposalCommentSchema,
    },
  );
}
