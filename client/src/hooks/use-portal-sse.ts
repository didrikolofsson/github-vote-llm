import { useEffect } from "react";

type UseSSEProps = {
  url: string;
  onMessage: (event: MessageEvent) => void;
  eventName?: string;
  enabled?: boolean;
};

const useSSE = ({
  url,
  onMessage,
  eventName = "event",
  enabled = true,
}: UseSSEProps) => {
  useEffect(() => {
    if (!enabled) return;

    const eventSource = new EventSource(url);
    eventSource.addEventListener(eventName, onMessage);
    eventSource.onerror = (event) =>
      console.error("Server-Sent Event error:", event);

    return () => {
      eventSource.removeEventListener(eventName, onMessage);
      eventSource.close();
    };
  }, [url]);
};

export default useSSE;
