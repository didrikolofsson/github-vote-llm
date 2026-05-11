import { Handle, Node, Position, type NodeProps } from "@xyflow/react";
import { ExternalLink, ThumbsUp } from "lucide-react";
import { Feature, Run, type FeatureBuildStatus } from "@/lib/api-schemas";

const STATUS_DOT: Record<NonNullable<FeatureBuildStatus>, string> = {
  pending: "bg-zinc-400",
  in_progress: "bg-amber-400",
  done: "bg-emerald-400",
  rejected: "bg-red-400",
  stuck: "bg-red-400",
};

const STATUS_LABEL: Record<NonNullable<FeatureBuildStatus>, string> = {
  pending: "Pending",
  in_progress: "In progress",
  stuck: "Stuck",
  done: "Done",
  rejected: "Rejected",
};

const RUN_DOT: Record<Run["status"], string> = {
  pending: "bg-amber-400",
  running: "bg-cyan-400 animate-pulse",
  completed: "bg-lime-400",
  failed: "bg-red-400",
  cancelled: "bg-zinc-400",
};

const RUN_LABEL: Record<Run["status"], string> = {
  pending: "Queued",
  running: "Running",
  completed: "Implemented",
  failed: "Run failed",
  cancelled: "Cancelled",
};

export type FeatureNodeData = Feature & { _latestRun?: Run };

type FeatureNodeProps = NodeProps<Node<FeatureNodeData>>;

export function FeatureNode({ data: feature, selected }: FeatureNodeProps) {
  const dotClass = STATUS_DOT[feature.build_status ?? "pending"];
  const statusLabel = STATUS_LABEL[feature.build_status ?? "pending"];
  const run = feature._latestRun;

  return (
    <div
      className={`
        w-[220px] rounded-lg border bg-card text-card-foreground shadow-sm
        transition-shadow duration-150
        ${selected ? "border-primary shadow-md ring-1 ring-primary/30" : "border-border hover:shadow-md"}
      `}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="!size-2.5 !border-2 !border-border !bg-background hover:!border-primary hover:!bg-primary/20 transition-colors"
      />

      <div className="p-3 flex flex-col gap-2">
        {/* Header: status dot + area badge */}
        <div className="flex items-center justify-between gap-1.5 min-w-0">
          <div className="flex items-center gap-1.5 min-w-0">
            <span className={`size-2 rounded-full shrink-0 ${dotClass}`} />
            <span className="text-[10px] text-muted-foreground uppercase tracking-wide font-medium truncate">
              {statusLabel}
            </span>
          </div>
          {feature.area && (
            <span className="text-[10px] font-mono bg-muted text-muted-foreground px-1.5 py-0.5 rounded shrink-0 max-w-[80px] truncate">
              {feature.area}
            </span>
          )}
        </div>

        {/* Title */}
        <p className="text-[13px] font-medium leading-snug line-clamp-2">
          {feature.title}
        </p>

        {/* Description preview */}
        {feature.description && (
          <p className="text-[11px] text-muted-foreground leading-snug line-clamp-2 -mt-1">
            {feature.description}
          </p>
        )}

        {/* Footer: votes + run badge */}
        <div className="flex items-center justify-between gap-1 text-xs text-muted-foreground pt-0.5 border-t border-border/50">
          <div className="flex items-center gap-1">
            <ThumbsUp className="size-3" />
            <span>{feature.vote_count ?? 0}</span>
          </div>
          {run && run.status !== "cancelled" && (
            <div className="flex items-center gap-1">
              {run.status === "completed" && run.pr_url ? (
                <a
                  href={run.pr_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={(e) => e.stopPropagation()}
                  className="flex items-center gap-1 text-info hover:underline"
                >
                  <ExternalLink className="size-3" />
                  <span className="text-[10px]">PR</span>
                </a>
              ) : (
                <>
                  <span
                    className={`size-1.5 rounded-full shrink-0 ${RUN_DOT[run.status]}`}
                  />
                  <span className="text-[10px]">{RUN_LABEL[run.status]}</span>
                </>
              )}
            </div>
          )}
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Right}
        className="!size-2.5 !border-2 !border-border !bg-background hover:!border-primary hover:!bg-primary/20 transition-colors"
      />
    </div>
  );
}
