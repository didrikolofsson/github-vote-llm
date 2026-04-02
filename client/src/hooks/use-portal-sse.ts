import { useEffect } from "react";

type UsePortalSSEProps = {
  orgSlug: string | null | undefined;
  repoName: string | null | undefined;
  repoId: number | null | undefined;
  onMessage: (event: MessageEvent) => void;
};

const usePortalSSE = ({
  orgSlug,
  repoName,
  repoId,
  onMessage,
}: UsePortalSSEProps) => {
  useEffect(() => {
    if (!orgSlug || !repoName || !repoId) return;

    const portalSSEUrl = `/v1/portal/${orgSlug}/${repoName}/events?repo_id=${repoId}`;
    const eventSource = new EventSource(portalSSEUrl);

    eventSource.addEventListener("event", onMessage);
    eventSource.onerror = (event) =>
      console.error("Error in portal SSE", event);

    return () => {
      eventSource.removeEventListener("event", onMessage);
      eventSource.close();
    };
  }, [orgSlug, repoName, repoId]);
};

export default usePortalSSE;
