import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

type PopupMessage =
  | {
      type: "github:complete";
      kind: "oauth" | "app_install";
      ok: boolean;
      org_id?: number;
    }
  | { type: string };

function isPopupMessage(data: unknown): data is PopupMessage {
  return (
    typeof data === "object" &&
    data !== null &&
    "type" in data &&
    typeof (data as { type: unknown }).type === "string"
  );
}

export function GithubPopupListener() {
  const queryClient = useQueryClient();

  useEffect(() => {
    function onMessage(e: MessageEvent) {
      if (e.origin !== window.location.origin) return;
      if (!isPopupMessage(e.data)) return;
      if (e.data.type !== "github:complete") return;
      if (!("ok" in e.data) || e.data.ok !== true) return;

      if (e.data.kind === "app_install") {
        if (typeof e.data.org_id === "number") {
          queryClient.invalidateQueries({
            queryKey: ["github-app-status", e.data.org_id],
          });
        } else {
          queryClient.invalidateQueries({ queryKey: ["github-app-status"] });
        }
      }

      if (e.data.kind === "oauth") {
        // No-op today, but keeps the UI self-healing as we add more GitHub connection queries.
        queryClient.invalidateQueries({ queryKey: ["github-account"] });
        queryClient.invalidateQueries({ queryKey: ["github-status"] });
      }
    }

    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [queryClient]);

  return null;
}

