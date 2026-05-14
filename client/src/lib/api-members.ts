import { z } from "zod";
import { OrganizationMemberRoleSchema } from "./api-schemas";
import { requestWithRefresh } from "./api-core";

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
