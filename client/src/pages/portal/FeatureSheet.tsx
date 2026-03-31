import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronUp } from "lucide-react";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { listPortalComments, type PortalFeature } from "@/lib/portal-api";
import { CommentForm } from "./CommentForm";

interface FeatureSheetProps {
  feature: PortalFeature | null;
  orgSlug: string;
  repoName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onVote: (featureId: number) => void;
}

const BUILD_STATUS_LABELS: Record<string, string> = {
  pending: "Pending",
  in_progress: "In Progress",
  stuck: "Stuck",
  done: "Done",
  rejected: "Rejected",
};

const BUILD_STATUS_COLORS: Record<string, string> = {
  pending: "bg-indigo-500/15 text-indigo-400",
  in_progress: "bg-amber-500/15 text-amber-400",
  stuck: "bg-red-500/15 text-red-400",
  done: "bg-emerald-500/15 text-emerald-400",
  rejected: "bg-muted text-muted-foreground",
};

export function FeatureSheet({
  feature,
  orgSlug,
  repoName,
  open,
  onOpenChange,
  onVote,
}: FeatureSheetProps) {
  const queryClient = useQueryClient();

  const { data: comments = [], isLoading: commentsLoading } = useQuery({
    queryKey: ["portal-comments", orgSlug, repoName, feature?.id],
    queryFn: () => listPortalComments(orgSlug, repoName, feature!.id),
    enabled: !!feature,
    staleTime: 10_000,
  });

  function handleCommentAdded() {
    queryClient.invalidateQueries({
      queryKey: ["portal-comments", orgSlug, repoName, feature?.id],
    });
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-lg overflow-y-auto flex flex-col gap-0 p-0">
        {feature && (
          <>
            <SheetHeader className="px-6 pt-6 pb-4">
              <div className="flex items-start gap-3">
                <button
                  type="button"
                  className={cn(
                    "shrink-0 flex flex-col items-center gap-0.5 rounded-lg px-2.5 py-2 transition-colors",
                    feature.has_voted
                      ? "bg-indigo-500/20 text-indigo-400 hover:bg-indigo-500/30"
                      : "bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground",
                  )}
                  onClick={() => onVote(feature.id)}
                  aria-label={feature.has_voted ? "Remove vote" : "Vote"}
                >
                  <ChevronUp className="size-4" />
                  <span className="text-xs font-semibold tabular-nums">{feature.vote_count}</span>
                </button>
                <div className="flex-1 min-w-0">
                  <SheetTitle className="text-base leading-snug">{feature.title}</SheetTitle>
                  <div className="mt-2 flex items-center gap-2 flex-wrap">
                    {feature.build_status && (
                      <span
                        className={cn(
                          "inline-flex items-center rounded-full px-2 py-0.5 text-xs",
                          BUILD_STATUS_COLORS[feature.build_status] ?? "bg-muted text-muted-foreground",
                        )}
                      >
                        {BUILD_STATUS_LABELS[feature.build_status] ?? feature.build_status}
                      </span>
                    )}
                    {feature.area && (
                      <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs bg-muted text-muted-foreground">
                        {feature.area}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </SheetHeader>

            {feature.description && (
              <div className="px-6 pb-4">
                <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                  {feature.description}
                </p>
              </div>
            )}

            <Separator />

            <div className="px-6 py-4 flex flex-col gap-4 flex-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Comments
              </p>

              {commentsLoading ? (
                <div className="flex flex-col gap-3">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : comments.length === 0 ? (
                <p className="text-sm text-muted-foreground">No comments yet. Be the first!</p>
              ) : (
                <ul className="flex flex-col gap-3">
                  {comments.map((c) => (
                    <li key={c.id} className="flex flex-col gap-1">
                      <div className="flex items-baseline gap-2">
                        <span className="text-xs font-medium text-foreground">{c.author_name}</span>
                        <span className="text-xs text-muted-foreground">
                          {new Date(c.created_at).toLocaleDateString("en-US", {
                            month: "short",
                            day: "numeric",
                          })}
                        </span>
                      </div>
                      <p className="text-sm text-muted-foreground">{c.body}</p>
                    </li>
                  ))}
                </ul>
              )}

              <Separator />

              <CommentForm
                orgSlug={orgSlug}
                repoName={repoName}
                featureId={feature.id}
                onCommentAdded={handleCommentAdded}
              />
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
