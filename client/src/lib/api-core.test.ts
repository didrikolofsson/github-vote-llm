import { describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { ApiError, formatApiError, setAccessToken } from "./api-core";

describe("ApiError", () => {
  it("sets name, message, status, and body", () => {
    const err = new ApiError("not found", 404, { error: "resource missing" });
    expect(err.name).toBe("ApiError");
    expect(err.message).toBe("not found");
    expect(err.status).toBe(404);
    expect(err.body).toEqual({ error: "resource missing" });
    expect(err).toBeInstanceOf(Error);
  });
});

describe("formatApiError", () => {
  it("extracts body.error from ApiError", () => {
    const err = new ApiError("fallback", 400, { error: "specific message" });
    expect(formatApiError(err)).toBe("specific message");
  });

  it("falls back to message when no body.error", () => {
    const err = new ApiError("the message", 500);
    expect(formatApiError(err)).toBe("the message");
  });

  it("formats ZodError", () => {
    const schema = z.object({ name: z.string() });
    const result = schema.safeParse({ name: 123 });
    expect(result.success).toBe(false);
    const msg = formatApiError((result as z.SafeParseError<unknown>).error);
    expect(msg).toContain("Validation failed");
    expect(msg).toContain("name");
  });

  it("returns message for generic Error", () => {
    expect(formatApiError(new Error("boom"))).toBe("boom");
  });

  it("stringifies unknown values", () => {
    expect(formatApiError("raw string")).toBe("raw string");
  });
});

describe("request", () => {
  it("throws ApiError on non-ok response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    const { request } = await import("./api-core");
    await expect(
      request("/some-path", { schema: z.unknown() }),
    ).rejects.toMatchObject({ status: 404, name: "ApiError" });
  });

  it("sends Authorization header when token is set", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    );
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("my-token");

    const { request } = await import("./api-core");
    await request("/test", { schema: z.object({ ok: z.boolean() }) });

    expect(mockFetch).toHaveBeenCalledWith(
      "/v1/test",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer my-token",
        }),
      }),
    );
  });

  it("skips Authorization header when skipAuth is true", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    );
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("my-token");

    const { request } = await import("./api-core");
    await request("/test", { schema: z.object({ ok: z.boolean() }), skipAuth: true });

    const [, init] = mockFetch.mock.calls[0];
    expect(init.headers).not.toHaveProperty("Authorization");
  });
});
