import { z } from 'zod';

// ─── Run ────────────────────────────────────────────────────────────────────

export const RunSchema = z.object({
  id: z.number(),
  owner: z.string(),
  repo: z.string(),
  issue_number: z.number(),
  status: z.enum(['pending', 'in_progress', 'done', 'failed', 'cancelled']),
  branch: z.string().nullable().optional(),
  pr_url: z.string().nullable().optional(),
  error: z.string().nullable().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const RunListSchema = z.array(RunSchema);

// ─── RepoConfig ──────────────────────────────────────────────────────────────

export const RepoConfigSchema = z.object({
  id: z.number(),
  owner: z.string(),
  repo: z.string(),
  label_approved: z.string(),
  label_in_progress: z.string(),
  label_done: z.string(),
  label_failed: z.string(),
  label_feature_request: z.string(),
  vote_threshold: z.number(),
  timeout_minutes: z.number(),
  max_budget_usd: z.number(),
  is_board_public: z.boolean(),
  updated_at: z.string(),
});

export const RepoConfigListSchema = z.array(RepoConfigSchema);

export const UpdateRepoConfigRequestSchema = z.object({
  label_approved: z.string().optional(),
  label_in_progress: z.string().optional(),
  label_done: z.string().optional(),
  label_failed: z.string().optional(),
  label_feature_request: z.string().optional(),
  vote_threshold: z.number().optional(),
  timeout_minutes: z.number().optional(),
  max_budget_usd: z.number().optional(),
  is_board_public: z.boolean().optional(),
  anthropic_api_key: z.string().optional(),
});

// ─── Proposal ────────────────────────────────────────────────────────────────

export const ProposalSchema = z.object({
  id: z.number(),
  title: z.string(),
  description: z.string().nullable().optional(),
  vote_count: z.number(),
  status: z.enum(['open', 'planned', 'done']),
  created_at: z.string(),
  updated_at: z.string(),
});

export const ProposalListSchema = z.array(ProposalSchema);

// ─── Organization ─────────────────────────────────────────────────────────────

export const OrganizationSchema = z.object({
  id: z.number(),
  name: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const OrganizationMemberSchema = z.object({
  user_id: z.number(),
  email: z.string(),
  role: z.string(),
});

export const OrganizationWithMembersSchema = OrganizationSchema.extend({
  members: z.array(OrganizationMemberSchema),
});

export const OrganizationListResponseSchema = z.object({
  organizations: z.array(OrganizationSchema),
});

// ─── ProposalComment ─────────────────────────────────────────────────────────

export const ProposalCommentSchema = z.object({
  id: z.number(),
  body: z.string(),
  author_name: z.string().nullable().optional(),
  created_at: z.string(),
});

export const ProposalCommentListSchema = z.array(ProposalCommentSchema);

// ─── Exported types ──────────────────────────────────────────────────────────

export type Run = z.infer<typeof RunSchema>;
export type RepoConfig = z.infer<typeof RepoConfigSchema>;
export type UpdateRepoConfigRequest = z.infer<typeof UpdateRepoConfigRequestSchema>;
export type Proposal = z.infer<typeof ProposalSchema>;
export type ProposalComment = z.infer<typeof ProposalCommentSchema>;
export type Organization = z.infer<typeof OrganizationSchema>;
export type OrganizationWithMembers = z.infer<typeof OrganizationWithMembersSchema>;
