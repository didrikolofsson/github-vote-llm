import { describe, expect, it, vi } from "vitest";
import { setAccessToken } from "./api-core";
import {
  getGithubAppInstallStatus,
  getGithubAppInstallURL,
  listGithubAppInstallationRepos,
} from "./api-github";

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("getGithubAppInstallURL", () => {
  it("returns install_url", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(ok({ install_url: "https://github.com/apps/my-app/installations/new" })),
    );
    setAccessToken("tok");
    const result = await getGithubAppInstallURL(1);
    expect(result.install_url).toContain("github.com/apps");
  });
});

describe("getGithubAppInstallStatus", () => {
  it("returns installed: false when not connected", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ installed: false })));
    setAccessToken("tok");
    const result = await getGithubAppInstallStatus(1);
    expect(result.installed).toBe(false);
  });

  it("returns installed: true with login when connected", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        ok({
          installed: true,
          installed_by_user_name: "alice",
          target_login: "acme",
          account_type: "Organization",
        }),
      ),
    );
    setAccessToken("tok");
    const result = await getGithubAppInstallStatus(1);
    expect(result.installed).toBe(true);
    expect(result.target_login).toBe("acme");
  });

  it("throws ApiError on 500", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "internal error" }), { status: 500 }),
      ),
    );
    setAccessToken("tok");
    await expect(getGithubAppInstallStatus(1)).rejects.toMatchObject({ status: 500 });
  });
});

describe("listGithubAppInstallationRepos", () => {
  it("returns repositories array", async () => {
    const repos = [
      { github_repository_id: 1, owner: "acme", name: "repo1", full_name: "acme/repo1", private: false },
    ];
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(ok({ repositories: repos, has_more: false })),
    );
    setAccessToken("tok");
    const result = await listGithubAppInstallationRepos(1);
    expect(result).toHaveLength(1);
    expect(result[0].full_name).toBe("acme/repo1");
  });

  it("includes page param in request URL", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(
      ok({ repositories: [], has_more: false }),
    );
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("tok");

    await listGithubAppInstallationRepos(1, 3);

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("page=3");
  });
});
