import {
  AppInstallURLResponseSchema,
  AppInstallationStatusSchema,
  GitHubInstallationRepoListResponseSchema,
} from "./api-schemas";
import { requestWithRefresh } from "./api-core";

export async function getGithubAppInstallURL(orgId: number) {
  return requestWithRefresh(`/organizations/${orgId}/github-app/install-url`, {
    schema: AppInstallURLResponseSchema,
  });
}

export async function getGithubAppInstallStatus(orgId: number) {
  return requestWithRefresh(`/organizations/${orgId}/github-app/status`, {
    schema: AppInstallationStatusSchema,
  });
}

export async function listGithubAppInstallationRepos(orgId: number, page = 1) {
  const data = await requestWithRefresh(
    `/organizations/${orgId}/github/repositories?page=${page}`,
    { schema: GitHubInstallationRepoListResponseSchema },
  );
  return data.repositories;
}
