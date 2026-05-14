import {
  OrganizationListResponseSchema,
  OrganizationSchema,
  OrganizationWithMembersSchema,
} from "./api-schemas";
import { requestWithRefresh } from "./api-core";

export async function listMyOrganizations() {
  const data = await requestWithRefresh("/organizations", {
    schema: OrganizationListResponseSchema,
  });
  return data.organizations;
}

export async function createOrganization(name: string, slug?: string) {
  return requestWithRefresh("/organizations", {
    method: "POST",
    body: JSON.stringify({ name, ...(slug ? { slug } : {}) }),
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

export async function updateOrganizationSlug(orgId: number, slug: string) {
  return requestWithRefresh(`/organizations/${orgId}/slug`, {
    method: "PATCH",
    body: JSON.stringify({ slug }),
    schema: OrganizationSchema,
  });
}
