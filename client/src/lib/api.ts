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
  FeatureCommentListResponseSchema,
  FeatureCommentSchema,
  FeatureListResponseSchema,
  FeatureSchema,
  OrganizationListResponseSchema,
  OrganizationMemberRoleSchema,
  OrganizationSchema,
  OrganizationWithMembersSchema,
  RepositoryListResponseSchema,
  RepositorySchema,
  RoadmapSchema,
} from "./api-schemas";

export type {
  Feature,
  FeatureComment,
  FeatureDependency,
  FeatureStatus,
  Organization,
  OrganizationMemberRole,
  OrganizationWithMembers,
  Repository,
  Roadmap,
} from "./api-schemas";

export const UserProfileSchema = z.object({
  id: z.number(),
  email: z.string(),
  username: z.string().nullable().optional(),
});
export type UserProfile = z.infer<typeof UserProfileSchema>;

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

// ─── Users ────────────────────────────────────────────────────────────────────

export async function getMe(): Promise<UserProfile> {
  return requestWithRefresh("/users/me", { schema: UserProfileSchema });
}

export async function updateUsername(username: string): Promise<UserProfile> {
  return requestWithRefresh("/users/me/username", {
    method: "PATCH",
    body: JSON.stringify({ username }),
    schema: UserProfileSchema,
  });
}

export async function deleteUser(userId: number): Promise<void> {
  await requestWithRefresh(`/users/${userId}`, {
    method: "DELETE",
    schema: z.void(),
  });
}

// ─── Organizations ────────────────────────────────────────────────────────────

export async function listMyOrganizations() {
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

export async function updateOrganization(orgId: number, name: string) {
  return requestWithRefresh(`/organizations/${orgId}`, {
    method: "PUT",
    body: JSON.stringify({ name }),
    schema: OrganizationSchema,
  });
}

// ─── GitHub ───────────────────────────────────────────────────────────────────

export const GitHubStatusSchema = z.object({
  connected: z.boolean(),
  login: z.string().nullable().optional(),
});
export async function getGitHubStatus() {
  return requestWithRefresh("/github/status", { schema: GitHubStatusSchema });
}

export const GitHubAuthorizeUrlSchema = z.object({
  authorize_url: z.string(),
});
export async function getGitHubAuthorizeUrl() {
  return requestWithRefresh("/github/authorize", {
    schema: GitHubAuthorizeUrlSchema,
  });
}

export async function disconnectGitHub(): Promise<void> {
  return requestWithRefresh("/github/connection", {
    method: "DELETE",
    schema: z.void(),
  });
}

export const AvailableRepositoriesSchema = z.object({
  repositories: z.array(z.object({ owner: z.string(), repo: z.string() })),
  has_more: z.boolean(),
});
export async function listAvailableRepositories(page = 1) {
  return requestWithRefresh(`/github/repositories?page=${page}`, {
    schema: AvailableRepositoriesSchema,
  });
}

// ─── Repositories ─────────────────────────────────────────────────────────────

export async function listOrgRepositories(orgId: number) {
  const data = await requestWithRefresh(`/organizations/${orgId}/repositories`, {
    schema: RepositoryListResponseSchema,
  });
  return data.repositories;
}

export async function addRepository(orgId: number, owner: string, repo: string) {
  return requestWithRefresh(`/organizations/${orgId}/repositories`, {
    method: "POST",
    body: JSON.stringify({ owner, repo }),
    schema: RepositorySchema,
  });
}

export async function removeRepository(orgId: number, repoId: number): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/repositories/${repoId}`, {
    method: "DELETE",
    schema: z.void(),
  });
}

// ─── Members ──────────────────────────────────────────────────────────────────

export const OrgMemberSchema = z.object({
  user_id: z.number(),
  email: z.string(),
  role: OrganizationMemberRoleSchema,
});
export type OrgMember = z.infer<typeof OrgMemberSchema>;

export async function listOrgMembers(orgId: number): Promise<OrgMember[]> {
  const data = await requestWithRefresh(`/organizations/${orgId}/members`, {
    schema: z.object({ members: z.array(OrgMemberSchema) }),
  });
  return data.members;
}

export async function inviteMember(orgId: number, email: string): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/members`, {
    method: "POST",
    body: JSON.stringify({ email }),
    schema: z.void(),
  });
}

export async function removeMember(orgId: number, userId: number): Promise<void> {
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

// ─── Features ─────────────────────────────────────────────────────────────────

export async function listFeatures(repoId: number) {
  const data = await requestWithRefresh(`/repositories/${repoId}/features`, {
    schema: FeatureListResponseSchema,
  });
  return data.features;
}

export async function getFeature(repoId: number, featureId: number) {
  return requestWithRefresh(`/repositories/${repoId}/features/${featureId}`, {
    schema: FeatureSchema,
  });
}

export async function createFeature(
  repoId: number,
  body: { title: string; description?: string },
) {
  return requestWithRefresh(`/repositories/${repoId}/features`, {
    method: "POST",
    body: JSON.stringify(body),
    schema: FeatureSchema,
  });
}

export async function updateFeatureTitle(
  repoId: number,
  featureId: number,
  title: string,
) {
  return requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/title`,
    {
      method: "PATCH",
      body: JSON.stringify({ title }),
      schema: FeatureSchema,
    },
  );
}

export async function updateFeatureStatus(
  repoId: number,
  featureId: number,
  status: string,
) {
  return requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/status`,
    {
      method: "PATCH",
      body: JSON.stringify({ status }),
      schema: FeatureSchema,
    },
  );
}

export async function updateFeatureArea(
  repoId: number,
  featureId: number,
  area: string | null,
) {
  return requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/area`,
    {
      method: "PATCH",
      body: JSON.stringify({ area }),
      schema: FeatureSchema,
    },
  );
}

export async function updateFeaturePosition(
  repoId: number,
  featureId: number,
  x: number | null,
  y: number | null,
  locked: boolean,
) {
  return requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/position`,
    {
      method: "PATCH",
      body: JSON.stringify({ x, y, locked }),
      schema: FeatureSchema,
    },
  );
}

export async function getRoadmap(repoId: number) {
  return requestWithRefresh(`/repositories/${repoId}/roadmap`, {
    schema: RoadmapSchema,
  });
}

export async function deleteFeature(repoId: number, featureId: number): Promise<void> {
  await requestWithRefresh(`/repositories/${repoId}/features/${featureId}`, {
    method: "DELETE",
    schema: z.void(),
  });
}

export async function addFeatureDependency(
  repoId: number,
  featureId: number,
  dependsOn: number,
): Promise<void> {
  await requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/dependencies`,
    {
      method: "POST",
      body: JSON.stringify({ depends_on: dependsOn }),
      schema: z.void(),
    },
  );
}

export async function removeFeatureDependency(
  repoId: number,
  featureId: number,
  dependsOn: number,
): Promise<void> {
  await requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/dependencies/${dependsOn}`,
    { method: "DELETE", schema: z.void() },
  );
}

export async function toggleFeatureVote(
  repoId: number,
  featureId: number,
  voterToken: string,
): Promise<{ vote_count: number }> {
  return requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/vote`,
    {
      method: "POST",
      body: JSON.stringify({ voter_token: voterToken }),
      schema: z.object({ vote_count: z.number() }),
    },
  );
}

export async function listFeatureComments(repoId: number, featureId: number) {
  const data = await requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/comments`,
    { schema: FeatureCommentListResponseSchema },
  );
  return data.comments;
}

export async function createFeatureComment(
  repoId: number,
  featureId: number,
  body: { body: string; author_name?: string },
) {
  return requestWithRefresh(
    `/repositories/${repoId}/features/${featureId}/comments`,
    {
      method: "POST",
      body: JSON.stringify(body),
      schema: FeatureCommentSchema,
    },
  );
}
