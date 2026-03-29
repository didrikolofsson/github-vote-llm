import { ChevronUp, MessageSquare } from "lucide-react";
import { cn } from "@/lib/utils";
import type { PortalFeature } from "@/lib/portal-api";

interface FeatureCardProps {
  feature: PortalFeature;
  onVote?: (featureId: number) => void;
  onClick: (featureId: number) => void;
  compact?: boolean;
}

const STATUS_COLORS: Record<string, string> = {
  open: "bg-indigo-500/15 text-indigo-400",
  planned: "bg-violet-500/15 text-violet-400",
  in_progress: "bg-amber-500/15 text-amber-400",
  done: "bg-emerald-500/15 text-emerald-400",
};

export function FeatureCard({ feature, onVote, onClick, compact = false }: FeatureCardProps) {
  return (
    <button
      type="button"
      className={cn(
        "w-full text-left rounded-xl border border-border bg-card hover:bg-accent/50 transition-colors",
        compact ? "px-4 py-3" : "px-4 py-4",
      )}
      onClick={() => onClick(feature.id)}
    >
      <div className="flex items-start gap-3">
        {/* Vote button */}
        {onVote && (
          <button
            type="button"
            className={cn(
              "shrink-0 flex flex-col items-center gap-0.5 rounded-lg px-2 py-1.5 transition-colors min-w-[40px]",
              feature.has_voted
                ? "bg-indigo-500/20 text-indigo-400 hover:bg-indigo-500/30"
                : "bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground",
            )}
            onClick={(e) => {
              e.stopPropagation();
              onVote(feature.id);
            }}
            aria-label={feature.has_voted ? "Remove vote" : "Vote for this feature"}
          >
            <ChevronUp className="size-3.5" />
            <span className="text-xs font-semibold tabular-nums">{feature.vote_count}</span>
          </button>
        )}

        {/* Content */}
        <div className="flex-1 min-w-0">
          <p className={cn("font-medium text-foreground leading-snug", compact ? "text-sm" : "text-[15px]")}>
            {feature.title}
          </p>
          {!compact && feature.description && (
            <p className="mt-1 text-sm text-muted-foreground line-clamp-2">
              {feature.description}
            </p>
          )}
          <div className="mt-2 flex items-center gap-2 flex-wrap">
            {feature.area && (
              <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs bg-muted text-muted-foreground">
                {feature.area}
              </span>
            )}
            {compact && (
              <span
                className={cn(
                  "inline-flex items-center rounded-full px-2 py-0.5 text-xs",
                  STATUS_COLORS[feature.status] ?? "bg-muted text-muted-foreground",
                )}
              >
                {feature.status.replace("_", " ")}
              </span>
            )}
          </div>
        </div>

        {/* Vote count (no-vote mode) */}
        {!onVote && (
          <div className="shrink-0 flex items-center gap-1 text-xs text-muted-foreground">
            <ChevronUp className="size-3" />
            <span>{feature.vote_count}</span>
          </div>
        )}
        <div className="shrink-0 flex items-center gap-1 text-xs text-muted-foreground">
          <MessageSquare className="size-3" />
        </div>
      </div>
    </button>
  );
}
