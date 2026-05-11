import { useEffect, useRef, useState } from "react";
import { getAccessToken } from "../lib/api";

export function useRunLogsSSE(runId: number | undefined): { lines: string[] } {
  const [lines, setLines] = useState<string[]>([]);
  const runIdRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (!runId) return;

    // Reset lines when switching to a different run.
    if (runIdRef.current !== runId) {
      runIdRef.current = runId;
      setLines([]);
    }

    const token = getAccessToken();
    if (!token) return;

    let active = true;
    const controller = new AbortController();

    (async () => {
      try {
        const response = await fetch(`/v1/runs/${runId}/logs`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });

        if (!response.ok || !response.body) return;

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (active) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const parts = buffer.split("\n");
          buffer = parts.pop() ?? "";

          for (const part of parts) {
            if (part.startsWith("data:")) {
              const line = part.slice(5).trimStart();
              setLines((prev) => [...prev, line]);
            }
          }
        }
      } catch {
        // AbortError on cleanup — expected.
      }
    })();

    return () => {
      active = false;
      controller.abort();
    };
  }, [runId]);

  return { lines };
}
