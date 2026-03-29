import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ThumbsUp } from "lucide-react";
import type { Feature, FeatureStatus } from "@/lib/api-schemas";

const STATUS_DOT: Record<FeatureStatus, string> = {
  open: "bg-zinc-400",
  planned: "bg-blue-400",
  in_progress: "bg-amber-400",
  done: "bg-emerald-400",
  rejected: "bg-red-400",
};

const STATUS_LABEL: Record<FeatureStatus, string> = {
  open: "Open",
  planned: "Planned",
  in_progress: "In progress",
  done: "Done",
  rejected: "Rejected",
};

export type FeatureNodeData = Feature & { selected?: boolean };

export function FeatureNode({ data, selected }: NodeProps) {
  const feature = data as FeatureNodeData;
  const dotClass = STATUS_DOT[feature.status as FeatureStatus] ?? "bg-zinc-400";
  const statusLabel = STATUS_LABEL[feature.status as FeatureStatus] ?? feature.status;

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

        {/* Footer: votes */}
        <div className="flex items-center gap-1 text-xs text-muted-foreground pt-0.5 border-t border-border/50">
          <ThumbsUp className="size-3" />
          <span>{feature.vote_count ?? 0}</span>
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
