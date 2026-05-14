import { describe, expect, it, vi } from "vitest";
import { ApiError, setAccessToken } from "./api-core";
import { cancelRun, createRun, deleteRun, getRun, listRepositoryRuns } from "./api-runs";

const run = {
  id: 1,
  prompt: "implement dark mode",
  feature_id: 5,
  status: "pending" as const,
  created_by_user_id: 1,
  created_at: "2024-01-01T00:00:00+00:00",
  completed_at: null,
  pr_url: null,
};

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("createRun", () => {
  it("returns created run", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(run)));
    setAccessToken("tok");
    const result = await createRun("implement dark mode", 5, 1);
    expect(result.id).toBe(1);
    expect(result.status).toBe("pending");
  });

  it("sends prompt and created_by_user_id in body", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(ok(run));
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("tok");

    await createRun("my prompt", 5, 42);

    const [, init] = mockFetch.mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({ prompt: "my prompt", created_by_user_id: 42 });
  });

  it("throws ApiError on 400", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "bad request" }), { status: 400 }),
      ),
    );
    setAccessToken("tok");
    await expect(createRun("", 5, 1)).rejects.toBeInstanceOf(ApiError);
  });
});

describe("getRun", () => {
  it("returns run by id", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(run)));
    setAccessToken("tok");
    const result = await getRun(1);
    expect(result.id).toBe(1);
    expect(result.prompt).toBe("implement dark mode");
  });

  it("throws ApiError on 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    await expect(getRun(99)).rejects.toMatchObject({ status: 404 });
  });
});

describe("listRepositoryRuns", () => {
  it("returns runs array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ runs: [run] })));
    setAccessToken("tok");
    const result = await listRepositoryRuns(10);
    expect(result).toHaveLength(1);
    expect(result[0].feature_id).toBe(5);
  });
});

describe("cancelRun", () => {
  it("resolves on success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(cancelRun(1)).resolves.not.toThrow();
  });
});

describe("deleteRun", () => {
  it("resolves on success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(deleteRun(1)).resolves.not.toThrow();
  });
});
