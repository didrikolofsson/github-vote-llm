import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "../lib/api";

export function useRunsSSE(repoId: number | undefined) {
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!repoId) return;

    const token = getAccessToken();
    if (!token) return;

    let active = true;
    const controller = new AbortController();

    (async () => {
      try {
        const response = await fetch(`/v1/repositories/${repoId}/runs/events`, {
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
          const lines = buffer.split("\n");
          buffer = lines.pop() ?? "";

          for (const line of lines) {
            if (line.startsWith("data:")) {
              const data = line.slice(5).trim();
              if (data === "run_updated") {
                queryClient.invalidateQueries({
                  queryKey: ["repositories", repoId, "runs"],
                });
                queryClient.invalidateQueries({
                  queryKey: ["repositories", repoId, "roadmap"],
                });
              }
            }
          }
        }
      } catch {
        // AbortError on cleanup or network failure — both are expected
      }
    })();

    return () => {
      active = false;
      controller.abort();
    };
  }, [repoId, queryClient]);
}
