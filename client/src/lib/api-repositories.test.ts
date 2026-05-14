import { describe, expect, it, vi } from "vitest";
import { ApiError, setAccessToken } from "./api-core";
import {
  addRepository,
  getRepoMeta,
  listOrgRepositories,
  removeRepository,
  updateRepositoryPortalPublic,
} from "./api-repositories";

const repo = {
  id: 10,
  owner: "acme",
  name: "my-repo",
  portal_public: false,
  created_at: "2024-01-01T00:00:00Z",
};

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("listOrgRepositories", () => {
  it("returns repositories array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ repositories: [repo] })));
    setAccessToken("tok");
    const result = await listOrgRepositories(1);
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("my-repo");
  });
});

describe("addRepository", () => {
  it("returns created repository", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(repo)));
    setAccessToken("tok");
    const result = await addRepository(1, "acme", "my-repo");
    expect(result.id).toBe(10);
    expect(result.owner).toBe("acme");
  });

  it("throws ApiError on 404 when repo not found on GitHub", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "repository not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    await expect(addRepository(1, "acme", "nonexistent")).rejects.toMatchObject({ status: 404 });
  });
});

describe("removeRepository", () => {
  it("resolves without error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(removeRepository(1, 10)).resolves.toBeUndefined();
  });
});

describe("updateRepositoryPortalPublic", () => {
  it("returns repository with updated portal_public", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ ...repo, portal_public: true })));
    setAccessToken("tok");
    const result = await updateRepositoryPortalPublic(10, true);
    expect(result.portal_public).toBe(true);
  });
});

describe("getRepoMeta", () => {
  it("returns repo metadata", async () => {
    const meta = { id: 10, description: "A repo", features: 5, implementations: 2, status: "active" };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(meta)));
    setAccessToken("tok");
    const result = await getRepoMeta(10);
    expect(result.features).toBe(5);
    expect(result.status).toBe("active");
  });

  it("throws ApiError on 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    await expect(getRepoMeta(99)).rejects.toBeInstanceOf(ApiError);
  });
});
