import { useEffect, useRef } from "react";

type UsePortalSSEProps = {
  orgSlug: string | null | undefined;
  repoName: string | null | undefined;
  repoId: number | null | undefined;
  onMessage: (data: string) => void;
};

const usePortalSSE = ({
  orgSlug,
  repoName,
  repoId,
  onMessage,
}: UsePortalSSEProps) => {
  const portalSSEUrl = `/v1/portal/${orgSlug}/${repoName}/events?repo_id=${repoId}`;

  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  useEffect(() => {
    if (!orgSlug || !repoName || !repoId) return;

    const eventSource = new EventSource(portalSSEUrl);
    eventSource.onmessage = (event) => {
      console.log("Portal SSE message received", event);
      onMessageRef.current?.(event.data);
    };
    eventSource.onerror = (event) =>
      console.error("Error in portal SSE", event);
    return () => eventSource.close();
  }, [orgSlug, repoName, repoId]);
};

export default usePortalSSE;
