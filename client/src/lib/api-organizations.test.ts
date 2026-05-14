import { describe, expect, it, vi } from "vitest";
import { ApiError, setAccessToken } from "./api-core";
import {
  createOrganization,
  listMyOrganizations,
  updateOrganization,
  updateOrganizationSlug,
} from "./api-organizations";

const org = {
  id: 1,
  name: "My Org",
  slug: "my-org",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("listMyOrganizations", () => {
  it("returns organizations array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ organizations: [org] })));
    setAccessToken("tok");
    const result = await listMyOrganizations();
    expect(result).toHaveLength(1);
    expect(result[0].slug).toBe("my-org");
  });

  it("returns empty array when no orgs", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ organizations: [] })));
    setAccessToken("tok");
    const result = await listMyOrganizations();
    expect(result).toEqual([]);
  });
});

describe("createOrganization", () => {
  it("returns created org with members", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(ok({ ...org, members: [{ user_id: 1, email: "a@b.com", role: "owner" }] })),
    );
    setAccessToken("tok");
    const result = await createOrganization("My Org");
    expect(result.name).toBe("My Org");
    expect(result.members).toHaveLength(1);
  });

  it("throws ApiError on conflict", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "slug already taken" }), { status: 409 }),
      ),
    );
    setAccessToken("tok");
    await expect(createOrganization("My Org", "my-org")).rejects.toMatchObject({ status: 409 });
  });
});

describe("updateOrganization", () => {
  it("returns updated org", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ ...org, name: "New Name" })));
    setAccessToken("tok");
    const result = await updateOrganization(1, "New Name");
    expect(result.name).toBe("New Name");
  });
});

describe("updateOrganizationSlug", () => {
  it("returns org with new slug", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ ...org, slug: "new-slug" })));
    setAccessToken("tok");
    const result = await updateOrganizationSlug(1, "new-slug");
    expect(result.slug).toBe("new-slug");
  });

  it("throws ApiError on 400 for invalid slug", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "invalid slug" }), { status: 400 }),
      ),
    );
    setAccessToken("tok");
    await expect(updateOrganizationSlug(1, "BAD SLUG!")).rejects.toBeInstanceOf(ApiError);
  });
});
