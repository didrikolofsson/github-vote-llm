import { describe, expect, it, vi } from "vitest";
import {
  createPortalComment,
  getPortalPage,
  listPortalComments,
  togglePortalVote,
} from "./portal-api";

const portalFeature = {
  id: 1,
  title: "Dark mode",
  description: "Add dark mode support",
  review_status: "approved" as const,
  build_status: null,
  area: null,
  vote_count: 3,
  has_voted: false,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

const portalPage = {
  org_slug: "acme",
  repo_owner: "acme",
  repo_name: "my-app",
  repo_id: 10,
  requests: [portalFeature],
  pending: [],
  in_progress: [],
  done: [],
};

const comment = {
  id: 1,
  feature_id: 1,
  body: "I need this!",
  author_name: "Bob",
  created_at: "2024-01-01T00:00:00Z",
};

function ok(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200 });
}

describe("getPortalPage", () => {
  it("returns portal page data", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(portalPage)));
    const result = await getPortalPage("acme", "my-app", "voter-token");
    expect(result.org_slug).toBe("acme");
    expect(result.requests).toHaveLength(1);
  });

  it("includes voter_token as query param", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(ok(portalPage));
    vi.stubGlobal("fetch", mockFetch);

    await getPortalPage("acme", "my-app", "my-token");

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("voter_token=my-token");
  });

  it("throws on 404 for unknown portal", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "not found" }), { status: 404 }),
      ),
    );
    await expect(getPortalPage("noop", "noop", "tok")).rejects.toThrow("not found");
  });
});

describe("togglePortalVote", () => {
  it("returns updated vote_count", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ vote_count: 4 })));
    const result = await togglePortalVote("acme", "my-app", 1, "tok", "I want this");
    expect(result.vote_count).toBe(4);
  });

  it("sends voter_token, reason, and urgency in body", async () => {
    const mockFetch = vi.fn().mockResolvedValueOnce(ok({ vote_count: 1 }));
    vi.stubGlobal("fetch", mockFetch);

    await togglePortalVote("acme", "my-app", 1, "tok", "blocking", "blocking");

    const [, init] = mockFetch.mock.calls[0];
    const body = JSON.parse(init.body);
    expect(body.voter_token).toBe("tok");
    expect(body.reason).toBe("blocking");
    expect(body.urgency).toBe("blocking");
  });
});

describe("listPortalComments", () => {
  it("returns comments array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ comments: [comment] })));
    const result = await listPortalComments("acme", "my-app", 1);
    expect(result).toHaveLength(1);
    expect(result[0].body).toBe("I need this!");
  });

  it("returns empty array when no comments", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok({ comments: [] })));
    const result = await listPortalComments("acme", "my-app", 1);
    expect(result).toEqual([]);
  });
});

describe("createPortalComment", () => {
  it("returns created comment", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(ok(comment)));
    const result = await createPortalComment("acme", "my-app", 1, "I need this!", "Bob");
    expect(result.body).toBe("I need this!");
    expect(result.author_name).toBe("Bob");
  });

  it("throws on server error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "server error" }), { status: 500 }),
      ),
    );
    await expect(
      createPortalComment("acme", "my-app", 1, "comment"),
    ).rejects.toThrow("server error");
  });
});
