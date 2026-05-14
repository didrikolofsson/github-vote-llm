import { describe, expect, it, vi } from "vitest";
import { ApiError } from "./api-core";
import {
  authorize,
  exchangeToken,
  refreshToken,
  revokeToken,
  signup,
} from "./api-auth";

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

function err(status: number, message: string) {
  return new Response(JSON.stringify({ error: message }), { status });
}

describe("authorize", () => {
  it("returns code and redirect_uri on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        ok({ code: "abc123", redirect_uri: "http://localhost/callback" }),
      ),
    );
    const result = await authorize({
      email: "a@b.com",
      password: "pw",
      code_challenge: "challenge",
      redirect_uri: "http://localhost/callback",
    });
    expect(result.code).toBe("abc123");
    expect(result.redirect_uri).toBe("http://localhost/callback");
  });

  it("throws ApiError on 401", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(err(401, "invalid credentials")));
    await expect(
      authorize({ email: "a@b.com", password: "bad", code_challenge: "c", redirect_uri: "r" }),
    ).rejects.toMatchObject({ status: 401, name: "ApiError" });
  });
});

describe("exchangeToken", () => {
  it("returns tokens on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        ok({
          access_token: "at",
          refresh_token: "rt",
          token_type: "bearer",
          expires_in: 3600,
        }),
      ),
    );
    const result = await exchangeToken({
      grant_type: "authorization_code",
      code: "abc",
      code_verifier: "ver",
      redirect_uri: "http://localhost/callback",
    });
    expect(result.access_token).toBe("at");
    expect(result.refresh_token).toBe("rt");
  });

  it("throws ApiError on 400", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(err(400, "invalid code")));
    await expect(
      exchangeToken({ grant_type: "authorization_code", code: "bad", code_verifier: "v", redirect_uri: "r" }),
    ).rejects.toBeInstanceOf(ApiError);
  });
});

describe("refreshToken", () => {
  it("returns new tokens", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        ok({ access_token: "new-at", token_type: "bearer", expires_in: 3600 }),
      ),
    );
    const result = await refreshToken("my-refresh-token");
    expect(result.access_token).toBe("new-at");
  });
});

describe("revokeToken", () => {
  it("resolves without error on success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    await expect(revokeToken("rt")).resolves.toBeUndefined();
  });
});

describe("signup", () => {
  it("returns new user on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        ok({ id: 1, email: "a@b.com", created_at: "2024-01-01", updated_at: "2024-01-01" }),
      ),
    );
    const result = await signup({ email: "a@b.com", password: "pw" });
    expect(result.id).toBe(1);
    expect(result.email).toBe("a@b.com");
  });

  it("throws ApiError when email is taken", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(err(409, "email already exists")));
    await expect(signup({ email: "taken@b.com", password: "pw" })).rejects.toBeInstanceOf(ApiError);
  });
});
