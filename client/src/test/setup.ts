import { afterEach, vi } from "vitest";
import { setAccessToken, setOnRefresh } from "../lib/api-core";

afterEach(() => {
  vi.unstubAllGlobals();
  setAccessToken(null);
  setOnRefresh(null);
});
