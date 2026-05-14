import { z } from "zod";
import {
  RepoMetaSchema,
  RepositoryListResponseSchema,
  RepositorySchema,
} from "./api-schemas";
import { requestWithRefresh } from "./api-core";

export async function listOrgRepositories(orgId: number) {
  const data = await requestWithRefresh(
    `/organizations/${orgId}/repositories`,
    { schema: RepositoryListResponseSchema },
  );
  return data.repositories;
}

export async function addRepository(
  orgId: number,
  owner: string,
  repo: string,
) {
  return requestWithRefresh(`/organizations/${orgId}/repositories`, {
    method: "POST",
    body: JSON.stringify({ owner, repo }),
    schema: RepositorySchema,
  });
}

export async function removeRepository(
  orgId: number,
  repoId: number,
): Promise<void> {
  await requestWithRefresh(`/organizations/${orgId}/repositories/${repoId}`, {
    method: "DELETE",
    schema: z.void(),
  });
}

export async function updateRepositoryPortalPublic(
  repoId: number,
  portalPublic: boolean,
) {
  return requestWithRefresh(`/repositories/${repoId}/portal`, {
    method: "PATCH",
    body: JSON.stringify({ portal_public: portalPublic }),
    schema: RepositorySchema,
  });
}

export async function getRepoMeta(repoId: number) {
  return requestWithRefresh(`/repositories/${repoId}/meta`, {
    method: "GET",
    schema: RepoMetaSchema,
  });
}
