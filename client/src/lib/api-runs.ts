import { z } from "zod";
import { RunListResponseSchema, RunSchema } from "./api-schemas";
import { requestWithRefresh } from "./api-core";

export async function createRun(
  prompt: string,
  featureId: number,
  createdByUserId: number,
) {
  return requestWithRefresh(`/features/${featureId}/runs`, {
    method: "POST",
    body: JSON.stringify({ prompt, created_by_user_id: createdByUserId }),
    schema: RunSchema,
  });
}

export async function listRepositoryRuns(repoId: number) {
  const data = await requestWithRefresh(`/repositories/${repoId}/runs`, {
    schema: RunListResponseSchema,
  });
  return data.runs;
}

export async function cancelRun(runId: number) {
  return requestWithRefresh(`/runs/${runId}/cancel`, {
    method: "POST",
    schema: z.unknown(),
  });
}

export async function deleteRun(runId: number) {
  return requestWithRefresh(`/runs/${runId}`, {
    method: "DELETE",
    schema: z.unknown(),
  });
}

export async function getRun(runId: number) {
  return requestWithRefresh(`/runs/${runId}`, {
    schema: RunSchema,
  });
}
