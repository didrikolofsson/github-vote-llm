import { describe, expect, it, vi } from "vitest";
import { ApiError, setAccessToken } from "./api-core";
import {
  addFeatureDependency,
  createFeature,
  createFeatureComment,
  deleteFeature,
  getFeature,
  getRoadmap,
  listFeatureComments,
  listFeatures,
  removeFeatureDependency,
  toggleFeatureVote,
  updateFeature,
  updateFeaturePosition,
} from "./api-features";

const feature = {
  id: 1,
  repository_id: 10,
  title: "Test feature",
  description: "A description",
  review_status: "pending" as const,
  build_status: null,
  area: null,
  roadmap_x: null,
  roadmap_y: null,
  roadmap_locked: false,
  vote_count: 0,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

const comment = {
  id: 1,
  feature_id: 1,
  body: "Great idea",
  author_name: "Alice",
  created_at: "2024-01-01T00:00:00Z",
};

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("listFeatures", () => {
  it("returns features array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ features: [feature] })));
    setAccessToken("tok");
    const result = await listFeatures(10);
    expect(result).toHaveLength(1);
    expect(result[0].title).toBe("Test feature");
  });
});

describe("getFeature", () => {
  it("returns a single feature", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(feature)));
    setAccessToken("tok");
    const result = await getFeature(10, 1);
    expect(result.id).toBe(1);
  });

  it("throws ApiError on 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    await expect(getFeature(10, 99)).rejects.toMatchObject({ status: 404 });
  });
});

describe("createFeature", () => {
  it("returns created feature", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(feature)));
    setAccessToken("tok");
    const result = await createFeature(10, { title: "Test feature" });
    expect(result.title).toBe("Test feature");
  });
});

describe("updateFeature", () => {
  it("returns updated feature", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(ok({ ...feature, review_status: "approved" })),
    );
    setAccessToken("tok");
    const result = await updateFeature(10, 1, { review_status: "approved" });
    expect(result.review_status).toBe("approved");
  });
});

describe("updateFeaturePosition", () => {
  it("sends x, y, locked in body", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(ok(feature));
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("tok");

    await updateFeaturePosition(10, 1, 100, 200, true);

    const [, init] = mockFetch.mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({ x: 100, y: 200, locked: true });
  });
});

describe("getRoadmap", () => {
  it("returns features and dependencies", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(ok({ features: [feature], dependencies: [] })),
    );
    setAccessToken("tok");
    const result = await getRoadmap(10);
    expect(result.features).toHaveLength(1);
    expect(result.dependencies).toEqual([]);
  });
});

describe("deleteFeature", () => {
  it("resolves without error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(deleteFeature(10, 1)).resolves.toBeUndefined();
  });
});

describe("addFeatureDependency", () => {
  it("resolves without error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(addFeatureDependency(10, 1, 2)).resolves.toBeUndefined();
  });

  it("sends depends_on in request body", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(new Response("", { status: 200 }));
    vi.stubGlobal("fetch", mockFetch);
    setAccessToken("tok");

    await addFeatureDependency(10, 1, 2);

    const [, init] = mockFetch.mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({ depends_on: 2 });
  });
});

describe("removeFeatureDependency", () => {
  it("resolves without error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response("", { status: 200 })));
    setAccessToken("tok");
    await expect(removeFeatureDependency(10, 1, 2)).resolves.toBeUndefined();
  });
});

describe("toggleFeatureVote", () => {
  it("returns vote_count", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ vote_count: 5 })));
    setAccessToken("tok");
    const result = await toggleFeatureVote(10, 1, "voter-token");
    expect(result.vote_count).toBe(5);
  });
});

describe("listFeatureComments", () => {
  it("returns comments array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ comments: [comment] })));
    setAccessToken("tok");
    const result = await listFeatureComments(10, 1);
    expect(result).toHaveLength(1);
    expect(result[0].body).toBe("Great idea");
  });
});

describe("createFeatureComment", () => {
  it("returns created comment", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(comment)));
    setAccessToken("tok");
    const result = await createFeatureComment(10, 1, { body: "Great idea", author_name: "Alice" });
    expect(result.body).toBe("Great idea");
  });

  it("throws ApiError on 404 when feature not found", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "feature not found" }), { status: 404 }),
      ),
    );
    setAccessToken("tok");
    await expect(
      createFeatureComment(10, 99, { body: "comment" }),
    ).rejects.toBeInstanceOf(ApiError);
  });
});
