import { describe, expect, it, vi } from "vitest";
import { setAccessToken } from "./api-core";
import {
  inviteMember,
  listOrgMembers,
  removeMember,
  updateMemberRole,
} from "./api-members";

const member = { user_id: 2, email: "b@b.com", role: "member" as const };

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("listOrgMembers", () => {
  it("returns members array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ members: [member] })));
    setAccessToken("tok");
    const result = await listOrgMembers(1);
    expect(result).toHaveLength(1);
    expect(result[0].email).toBe("b@b.com");
    expect(result[0].role).toBe("member");
  });
});

describe("inviteMember", () => {
  it("resolves without error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(inviteMember(1, "new@b.com")).resolves.toBeUndefined();
  });

  it("throws on 404 when org not found", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "org not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    await expect(inviteMember(99, "a@b.com")).rejects.toMatchObject({ status: 404 });
  });
});

describe("removeMember", () => {
  it("resolves without error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(removeMember(1, 2)).resolves.toBeUndefined();
  });
});

describe("updateMemberRole", () => {
  it("resolves without error for owner role", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(updateMemberRole(1, 2, "owner")).resolves.toBeUndefined();
  });

  it("sends correct role in request body", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(new Response("", { status: 200 }));
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("tok");

    await updateMemberRole(1, 2, "member");

    const [, init] = mockFetch.mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({ role: "member" });
  });
});
