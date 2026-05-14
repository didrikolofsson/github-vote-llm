import { z } from "zod";
import {
  FeatureCommentListResponseSchema,
  FeatureCommentSchema,
  FeatureListResponseSchema,
  FeatureSchema,
  RoadmapSchema,
} from "./api-schemas";
import { requestWithRefresh } from "./api-core";

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

export async function updateFeature(
  repoId: number,
  featureId: number,
  patch: {
    title?: string;
    description?: string;
    status?: string;
    build_status?: string | null;
    review_status?: string;
    area?: string;
  },
) {
  return requestWithRefresh(`/repositories/${repoId}/features/${featureId}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
    schema: FeatureSchema,
  });
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

export async function deleteFeature(
  repoId: number,
  featureId: number,
): Promise<void> {
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
