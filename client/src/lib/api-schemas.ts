import { z } from "zod";

// ─── Organization ─────────────────────────────────────────────────────────────

export const OrganizationMemberRoleSchema = z.enum(["owner", "member"]);

export const OrganizationSchema = z.object({
  id: z.number(),
  name: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const OrganizationMemberSchema = z.object({
  user_id: z.number(),
  email: z.string(),
  role: OrganizationMemberRoleSchema,
});

export const OrganizationWithMembersSchema = OrganizationSchema.extend({
  members: z.array(OrganizationMemberSchema),
});

export const OrganizationListResponseSchema = z.object({
  organizations: z.array(OrganizationSchema),
});

// ─── Repository ───────────────────────────────────────────────────────────────

export const RepositorySchema = z.object({
  id: z.number(),
  owner: z.string(),
  name: z.string(),
  created_at: z.string(),
});

export const RepositoryListResponseSchema = z.object({
  repositories: z.array(RepositorySchema),
});

// ─── Feature ──────────────────────────────────────────────────────────────────

export const FeatureStatusSchema = z.enum([
  "open",
  "planned",
  "in_progress",
  "done",
  "rejected",
]);

export const FeatureSchema = z.object({
  id: z.number(),
  repository_id: z.number(),
  title: z.string(),
  description: z.string(),
  status: FeatureStatusSchema,
  area: z.string().nullable().optional(),
  roadmap_x: z.number().nullable().optional(),
  roadmap_y: z.number().nullable().optional(),
  roadmap_locked: z.boolean(),
  vote_count: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const FeatureListResponseSchema = z.object({
  features: z.array(FeatureSchema),
});

export const FeatureCommentSchema = z.object({
  id: z.number(),
  feature_id: z.number(),
  body: z.string(),
  author_name: z.string(),
  created_at: z.string(),
});

export const FeatureCommentListResponseSchema = z.object({
  comments: z.array(FeatureCommentSchema),
});

export const FeatureDependencySchema = z.object({
  feature_id: z.number(),
  depends_on: z.number(),
});

export const RoadmapSchema = z.object({
  features: z.array(FeatureSchema),
  dependencies: z.array(FeatureDependencySchema),
});

// ─── Exported types ───────────────────────────────────────────────────────────

export type Organization = z.infer<typeof OrganizationSchema>;
export type OrganizationWithMembers = z.infer<typeof OrganizationWithMembersSchema>;
export type OrganizationMemberRole = z.infer<typeof OrganizationMemberRoleSchema>;
export type Repository = z.infer<typeof RepositorySchema>;
export type Feature = z.infer<typeof FeatureSchema>;
export type FeatureStatus = z.infer<typeof FeatureStatusSchema>;
export type FeatureComment = z.infer<typeof FeatureCommentSchema>;
export type FeatureDependency = z.infer<typeof FeatureDependencySchema>;
export type Roadmap = z.infer<typeof RoadmapSchema>;
