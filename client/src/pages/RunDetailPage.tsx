import { Badge, type BadgeColor } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cancelRun, deleteRun, getFeature, getRun } from "@/lib/api";
import type { RunStatus } from "@/lib/api-schemas";
import { useRunLogsSSE } from "@/hooks/use-run-logs-sse";
import { cn } from "@/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowUpRight,
  Bot,
  Clock3,
  Terminal,
  Trash2,
  X,
} from "lucide-react";
import { useEffect, useRef } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

const RUN_STATUS_META: Record<
  RunStatus,
  { label: string; color: BadgeColor; dot: string }
> = {
  pending: { label: "Pending", color: "amber", dot: "bg-amber-400" },
  running: { label: "Running", color: "cyan", dot: "bg-cyan-400 animate-pulse" },
  completed: { label: "Completed", color: "lime", dot: "bg-lime-400" },
  failed: { label: "Failed", color: "red", dot: "bg-red-400" },
  cancelled: { label: "Cancelled", color: "zinc", dot: "bg-zinc-400" },
};

function formatDate(value: string | null | undefined) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function LogLine({ line }: { line: string }) {
  const isStderr = line.startsWith("[stderr]");
  const text = line.replace(/^\[(stdout|stderr)\]\s?/, "");
  return (
    <div className={cn("leading-5", isStderr ? "text-amber-400/80" : "text-emerald-400/90")}>
      <span className="select-none text-muted-foreground/40 mr-2 text-[10px]">
        {isStderr ? "err" : "out"}
      </span>
      {text}
    </div>
  );
}

export default function RunDetailPage() {
  const { repoId, runId } = useParams<{ repoId: string; runId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const logBottomRef = useRef<HTMLDivElement>(null);

  const repoIdNum = repoId ? parseInt(repoId, 10) : undefined;
  const runIdNum = runId ? parseInt(runId, 10) : undefined;

  const { data: run, isLoading } = useQuery({
    queryKey: ["run", runIdNum],
    queryFn: () => getRun(runIdNum!),
    enabled: !!runIdNum,
    refetchInterval: (query) => {
      const s = query.state.data?.status;
      return s === "pending" || s === "running" ? 3000 : false;
    },
  });

  const { data: feature } = useQuery({
    queryKey: ["repositories", repoIdNum, "features", run?.feature_id],
    queryFn: () => getFeature(repoIdNum!, run!.feature_id),
    enabled: !!repoIdNum && !!run?.feature_id,
  });

  const { lines } = useRunLogsSSE(runIdNum);

  // Auto-scroll to bottom as new lines arrive.
  useEffect(() => {
    logBottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines.length]);

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["run", runIdNum] });
    queryClient.invalidateQueries({ queryKey: ["repositories", repoIdNum, "runs"] });
  }

  const cancelMutation = useMutation({
    mutationFn: () => cancelRun(runIdNum!),
    onSuccess: invalidate,
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteRun(runIdNum!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repositories", repoIdNum, "runs"] });
      navigate(`/repositories/${repoIdNum}?tab=runs`);
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64 text-sm text-muted-foreground">
        Loading run…
      </div>
    );
  }

  if (!run) return null;

  const meta = RUN_STATUS_META[run.status];
  const canCancel = run.status === "pending" || run.status === "running";
  const canDelete = run.status === "cancelled" || run.status === "failed";
  const isActive = run.status === "pending" || run.status === "running";

  return (
    <div className="animate-slide-up px-8 py-8 w-full max-w-[1280px] mx-auto flex flex-col gap-6">
      {/* Back nav */}
      <Link
        to={`/repositories/${repoIdNum}?tab=runs`}
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="size-3.5" />
        Back to runs
      </Link>

      {/* Header card */}
      <div className="relative overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
        <div className="absolute -right-16 -top-24 size-56 rounded-full bg-info/10 blur-3xl" />
        <div className="relative flex flex-col gap-4 p-5 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <Badge color={meta.color} small>
                <span className={cn("size-1.5 rounded-full", meta.dot)} />
                {meta.label}
              </Badge>
              <span className="font-mono text-xs text-muted-foreground">
                RUN-{String(run.id).padStart(4, "0")}
              </span>
            </div>
            <h1 className="text-xl font-semibold tracking-tight">
              {feature?.title ?? `Feature #${run.feature_id}`}
            </h1>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground line-clamp-2">
              {run.prompt}
            </p>
          </div>

          <div className="flex shrink-0 flex-col gap-2 lg:items-end">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Clock3 className="size-3.5" />
              <span>Started {formatDate(run.created_at)}</span>
            </div>
            {run.completed_at && (
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Clock3 className="size-3.5" />
                <span>Finished {formatDate(run.completed_at)}</span>
              </div>
            )}
            <div className="flex gap-2 mt-1">
              {run.status === "completed" && run.pr_url && (
                <Button variant="outline" size="sm" asChild>
                  <a href={run.pr_url} target="_blank" rel="noopener noreferrer">
                    View pull request
                    <ArrowUpRight data-icon="inline-end" />
                  </a>
                </Button>
              )}
              {canCancel && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => cancelMutation.mutate()}
                  disabled={cancelMutation.isPending}
                >
                  <X data-icon="inline-start" />
                  {cancelMutation.isPending ? "Cancelling…" : "Cancel run"}
                </Button>
              )}
              {canDelete && (
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => deleteMutation.mutate()}
                  disabled={deleteMutation.isPending}
                >
                  <Trash2 data-icon="inline-start" />
                  {deleteMutation.isPending ? "Deleting…" : "Delete run"}
                </Button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Log output */}
      <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
        <div className="flex items-center gap-2 border-b border-border/70 px-4 py-3 text-xs font-medium text-muted-foreground">
          <Terminal className="size-3.5" />
          <span className="uppercase tracking-[0.18em]">Agent output</span>
          {isActive && (
            <span className="ml-auto flex items-center gap-1.5 text-cyan-400">
              <span className="size-1.5 rounded-full bg-cyan-400 animate-pulse" />
              Live
            </span>
          )}
        </div>
        <div className="h-[calc(100vh-420px)] min-h-[300px] overflow-y-auto bg-zinc-950 p-4">
          {lines.length === 0 ? (
            <p className="text-xs text-muted-foreground/50 font-mono">
              {run.status === "pending"
                ? "Waiting for run to start…"
                : run.status === "running"
                  ? "Agent is starting up…"
                  : "No output recorded for this run."}
            </p>
          ) : (
            <div className="font-mono text-xs space-y-0.5">
              {lines.map((line, i) => (
                <LogLine key={i} line={line} />
              ))}
              {!isActive && (
                <div className="mt-2 text-muted-foreground/40 text-[10px] font-mono border-t border-border/20 pt-2">
                  — run ended —
                </div>
              )}
            </div>
          )}
          <div ref={logBottomRef} />
        </div>
      </div>

      {/* Implementation prompt detail */}
      <div className="rounded-xl border border-border bg-card shadow-sm p-4">
        <div className="mb-2 flex items-center gap-2 text-xs font-medium text-muted-foreground">
          <Bot className="size-3.5" />
          Implementation prompt
        </div>
        <p className="text-sm leading-6 text-muted-foreground whitespace-pre-wrap">
          {run.prompt}
        </p>
      </div>
    </div>
  );
}
