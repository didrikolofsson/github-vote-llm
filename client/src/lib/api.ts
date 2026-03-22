/**
 * API client for vote-llm backend.
 * Uses fetch with Bearer token for protected routes.
 * Validates responses with Zod schemas.
 */

import type { z } from 'zod';

const BASE = '/v1';

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
    this.name = 'ApiError';
  }
}

async function request<T>(
  path: string,
  options: RequestInit & { schema?: z.ZodType<T>; skipAuth?: boolean } = {},
): Promise<T> {
  const { schema, skipAuth, ...init } = options;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init.headers as Record<string, string>),
  };

  if (!skipAuth && accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`;
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
    const errBody = typeof data === 'object' && data !== null && 'error' in data
      ? (data as { error: string }).error
      : data;
    throw new ApiError(
      typeof errBody === 'string' ? errBody : `Request failed: ${res.status}`,
      res.status,
      data,
    );
  }

  if (schema && data !== undefined) {
    return schema.parse(data) as T;
  }

  return data as T;
}

// ─── Auth API (no token required) ─────────────────────────────────────────────

import {
  AuthorizeResponseSchema,
  TokenResponseSchema,
  SignupResponseSchema,
} from './auth-schemas';

export async function authorize(params: {
  email: string;
  password: string;
  code_challenge: string;
  redirect_uri: string;
}) {
  return request('/auth/authorize', {
    method: 'POST',
    body: JSON.stringify(params),
    schema: AuthorizeResponseSchema,
    skipAuth: true,
  });
}

export async function exchangeToken(params: {
  grant_type: 'authorization_code';
  code: string;
  code_verifier: string;
  redirect_uri: string;
}) {
  return request('/auth/token', {
    method: 'POST',
    body: JSON.stringify(params),
    schema: TokenResponseSchema,
    skipAuth: true,
  });
}

export async function refreshToken(refresh_token: string) {
  return request('/auth/token', {
    method: 'POST',
    body: JSON.stringify({ grant_type: 'refresh_token', refresh_token }),
    schema: TokenResponseSchema,
    skipAuth: true,
  });
}

export async function revokeToken(refresh_token: string) {
  return request('/auth/revoke', {
    method: 'POST',
    body: JSON.stringify({ refresh_token }),
    skipAuth: true,
  });
}

export async function signup(params: { email: string; password: string }) {
  return request('/users/signup', {
    method: 'POST',
    body: JSON.stringify(params),
    schema: SignupResponseSchema,
    skipAuth: true,
  });
}

// ─── App API (protected, requires token) ─────────────────────────────────────

import {
  RunSchema,
  RunListSchema,
  RepoConfigSchema,
  RepoConfigListSchema,
  UpdateRepoConfigRequestSchema,
  ProposalSchema,
  ProposalListSchema,
  ProposalCommentSchema,
  ProposalCommentListSchema,
  OrganizationListResponseSchema,
  OrganizationWithMembersSchema,
} from './api-schemas';

import type { Proposal, Organization, RepoConfig, Run, ProposalComment } from './api-schemas';
export type { Run, RepoConfig, UpdateRepoConfigRequest, Proposal, ProposalComment, Organization, OrganizationWithMembers } from './api-schemas';

let onRefresh: (() => Promise<string | null>) | null = null;

export function setOnRefresh(fn: (() => Promise<string | null>) | null): void {
  onRefresh = fn;
}

async function requestWithRefresh<T>(
  path: string,
  options: RequestInit & { schema?: z.ZodType<T>; skipAuth?: boolean },
): Promise<T> {
  try {
    return await request<T>(path, options);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401 && onRefresh && !options.skipAuth) {
      const newToken = await onRefresh();
      if (newToken) {
        setAccessToken(newToken);
        return request<T>(path, options);
      }
    }
    throw err;
  }
}

export async function listMyOrganizations(): Promise<Organization[]> {
  const data = await requestWithRefresh('/organizations', {
    schema: OrganizationListResponseSchema,
  });
  return data.organizations;
}

export async function createOrganization(name: string) {
  return requestWithRefresh('/organizations', {
    method: 'POST',
    body: JSON.stringify({ name }),
    schema: OrganizationWithMembersSchema,
  });
}

// GitHub connection
export async function getGitHubStatus(): Promise<{ connected: boolean; login?: string }> {
  return requestWithRefresh('/auth/github/status', {}) as Promise<{ connected: boolean; login?: string }>;
}

export async function getGitHubAuthorizeUrl(): Promise<{ authorize_url: string }> {
  return requestWithRefresh('/auth/github/authorize', {}) as Promise<{ authorize_url: string }>;
}

// Organization repositories
export type OrgRepository = { owner: string; repo: string; created_at?: string };
export async function listOrgRepositories(orgId: number): Promise<OrgRepository[]> {
  const data = await requestWithRefresh(`/organizations/${orgId}/repositories`, {}) as { repositories: OrgRepository[] };
  return data.repositories;
}

export async function listAvailableRepositories(orgId: number, page = 1): Promise<{ repositories: OrgRepository[]; has_more: boolean }> {
  return requestWithRefresh(`/organizations/${orgId}/repositories/available?page=${page}`, {}) as Promise<{ repositories: OrgRepository[]; has_more: boolean }>;
}

export async function addRepository(orgId: number, owner: string, repo: string): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/repositories`, {
    method: 'POST',
    body: JSON.stringify({ owner, repo }),
  });
}

export async function removeRepository(orgId: number, owner: string, repo: string): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/repositories/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`, { method: 'DELETE' });
}

// Organization members
export type OrgMember = { user_id: number; email: string; role: string };
export async function listOrgMembers(orgId: number): Promise<OrgMember[]> {
  const data = await requestWithRefresh(`/organizations/${orgId}/members`, {}) as { members: OrgMember[] };
  return data.members;
}

export async function inviteMember(orgId: number, email: string): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/members`, {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
}

export async function removeMember(orgId: number, userId: number): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/members/${userId}`, { method: 'DELETE' });
}

export async function updateMemberRole(orgId: number, userId: number, role: 'owner' | 'member'): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/members/${userId}`, {
    method: 'PATCH',
    body: JSON.stringify({ role }),
  });
}

export async function listRepos(): Promise<RepoConfig[]> {
  const data = await requestWithRefresh('/repos', {
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
    method: 'PUT',
    body: JSON.stringify(body),
    schema: RepoConfigSchema,
  });
}

export async function deleteRepoConfig(owner: string, repo: string) {
  await requestWithRefresh(`/repos/${owner}/${repo}/config`, {
    method: 'DELETE',
  });
}

export async function listRuns(): Promise<Run[]> {
  const data = await requestWithRefresh('/runs', {
    schema: RunListSchema,
  });
  return data;
}

export async function getRun(id: number) {
  return requestWithRefresh(`/runs/${id}`, {
    schema: RunSchema,
  });
}

export async function cancelRun(id: number) {
  await requestWithRefresh(`/runs/${id}/cancel`, { method: 'POST' });
}

export async function retryRun(id: number) {
  await requestWithRefresh(`/runs/${id}/retry`, { method: 'POST' });
}

export async function listRoadmapItems(owner: string, repo: string): Promise<Proposal[]> {
  const data = await requestWithRefresh(`/repos/${owner}/${repo}/roadmap`, {
    schema: ProposalListSchema,
  });
  return data;
}

export async function updateProposalStatus(
  owner: string,
  repo: string,
  id: number,
  body: { status: Proposal['status'] },
) {
  return requestWithRefresh(`/repos/${owner}/${repo}/proposals/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    schema: ProposalSchema,
  });
}

export async function listBoardProposals(owner: string, repo: string): Promise<Proposal[]> {
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
    method: 'POST',
    body: JSON.stringify(body),
    schema: ProposalSchema,
  });
}

export async function voteProposal(owner: string, repo: string, id: number): Promise<Proposal> {
  return requestWithRefresh(`/board/${owner}/${repo}/proposals/${id}/vote`, {
    method: 'POST',
    schema: ProposalSchema,
  });
}

export async function listBoardComments(
  owner: string,
  repo: string,
  id: number,
): Promise<ProposalComment[]> {
  const data = await requestWithRefresh(`/board/${owner}/${repo}/proposals/${id}/comments`, {
    schema: ProposalCommentListSchema,
  });
  return data;
}

export async function createBoardComment(
  owner: string,
  repo: string,
  id: number,
  body: { body: string; author_name?: string },
) {
  return requestWithRefresh(`/board/${owner}/${repo}/proposals/${id}/comments`, {
    method: 'POST',
    body: JSON.stringify(body),
    schema: ProposalCommentSchema,
  });
}
