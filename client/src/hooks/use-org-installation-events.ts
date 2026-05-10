import { getAccessToken } from "@/lib/api";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { toast } from "sonner";

type InstallationEvent =
  | "installation_active"
  | "installation_suspended"
  | "installation_removed";

const TOAST_BY_EVENT: Record<InstallationEvent, () => void> = {
  installation_active: () => toast.success("GitHub App connected"),
  installation_suspended: () => toast.warning("GitHub App suspended"),
  installation_removed: () => toast.info("GitHub App removed"),
};

export function useOrgInstallationEvents(orgId: number | undefined) {
  const queryClient = useQueryClient();
  const lastEventRef = useRef<InstallationEvent | null>(null);

  useEffect(() => {
    if (orgId == null) return;
    const token = getAccessToken();
    if (!token) return;

    const url = `/v1/organizations/${orgId}/events?access_token=${encodeURIComponent(token)}`;
    const es = new EventSource(url);

    const onMessage = (event: MessageEvent) => {
      const data = event.data as string;
      if (data === "connected") return;

      queryClient.invalidateQueries({ queryKey: ["github-app-status", orgId] });

      if (data in TOAST_BY_EVENT) {
        const evt = data as InstallationEvent;
        if (lastEventRef.current !== evt) {
          TOAST_BY_EVENT[evt]();
          lastEventRef.current = evt;
        }
      }
    };

    es.addEventListener("event", onMessage);
    es.onerror = (e) => console.error("Org installation SSE error:", e);

    return () => {
      es.removeEventListener("event", onMessage);
      es.close();
      lastEventRef.current = null;
    };
  }, [orgId, queryClient]);
}
