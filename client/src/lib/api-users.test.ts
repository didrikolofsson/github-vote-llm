import { describe, expect, it, vi } from "vitest";
import { ApiError, setAccessToken, setOnRefresh } from "./api-core";
import { deleteUser, getMe, updateUsername } from "./api-users";

const profile = { id: 1, email: "a@b.com", username: null };

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("getMe", () => {
  it("returns user profile", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(profile)));
    setAccessToken("tok");
    const result = await getMe();
    expect(result).toEqual(profile);
  });

  it("sends Authorization header", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(ok(profile));
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("my-token");

    await getMe();

    const [, init] = mockFetch.mock.calls[0];
    expect(init.headers.Authorization).toBe("Bearer my-token");
  });

  it("throws ApiError on 401 when no refresh handler", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 }),
      ),
    );
    setAccessToken("bad-token");
    await expect(getMe()).rejects.toMatchObject({ status: 401, name: "ApiError" });
  });

  it("refreshes token on 401 and retries", async () => {
    const mockFetch = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 }))
      .mockResolvedValueOnce(ok(profile));
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("old-token");
    setOnRefresh(vi.fn().mockResolvedValue("new-token"));

    const result = await getMe();
    expect(result.id).toBe(1);
    expect(mockFetch).toHaveBeenCalledTimes(2);
  });

  it("throws if refresh returns null", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "Unauthorized" }), { status: 401 }),
      ),
    );
    setAccessToken("bad-token");
    setOnRefresh(vi.fn().mockResolvedValue(null));
    await expect(getMe()).rejects.toBeInstanceOf(ApiError);
  });
});

describe("updateUsername", () => {
  it("returns updated profile", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ ...profile, username: "newname" })));
    setAccessToken("tok");
    const result = await updateUsername("newname");
    expect(result.username).toBe("newname");
  });
});

describe("deleteUser", () => {
  it("resolves without error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(deleteUser(1)).resolves.toBeUndefined();
  });

  it("throws ApiError on 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    await expect(deleteUser(99)).rejects.toMatchObject({ status: 404 });
  });
});
