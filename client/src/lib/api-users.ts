import { z } from "zod";
import { requestWithRefresh } from "./api-core";

export const UserProfileSchema = z.object({
  id: z.number(),
  email: z.string(),
  username: z.string().nullable().optional(),
});
export type UserProfile = z.infer<typeof UserProfileSchema>;

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
