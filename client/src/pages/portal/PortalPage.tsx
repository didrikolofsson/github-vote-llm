import { useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { GitFork } from "lucide-react";
import {
  getPortalPage,
  togglePortalVote,
  type PortalFeature,
  type PortalPage,
} from "@/lib/portal-api";
import { Skeleton } from "@/components/ui/skeleton";
import { ProposalsBoard } from "./ProposalsBoard";
import { RoadmapColumns } from "./RoadmapColumns";
import { RecentlyShipped } from "./RecentlyShipped";
import { FeatureSheet } from "./FeatureSheet";
import useSSE from "@/hooks/use-portal-sse";

function getOrCreateVoterToken(): string {
  const key = "voter_token";
  const existing = localStorage.getItem(key);
  if (existing) return existing;
  const token = crypto.randomUUID();
  localStorage.setItem(key, token);
  return token;
}

export default function PortalPage() {
  const { orgSlug, repoName } = useParams<{
    orgSlug: string;
    repoName: string;
    repoId: string;
  }>();
  const queryClient = useQueryClient();

  const voterToken = useMemo(() => getOrCreateVoterToken(), []);

  const queryKey = ["portal", orgSlug, repoName, voterToken];

  const { data, isLoading, isError, error } = useQuery({
    queryKey,
    queryFn: () => getPortalPage(orgSlug!, repoName!, voterToken),
    enabled: !!orgSlug && !!repoName,
  });

  useSSE({
    url: `/v1/portal/${orgSlug}/${repoName}/events?repo_id=${data?.repo_id}`,
    onMessage: () => queryClient.refetchQueries({ queryKey }),
    enabled: !!orgSlug && !!repoName && !!data?.repo_id,
  });

  const [selectedFeatureId, setSelectedFeatureId] = useState<number | null>(
    null,
  );
  const selectedFeature = useMemo(() => {
    if (selectedFeatureId == null || !data) return null;
    const all = [
      ...data.requests,
      ...data.pending,
      ...data.in_progress,
      ...data.done,
    ].sort((a, b) => b.updated_at.localeCompare(a.updated_at));
    return all.find((f) => f.id === selectedFeatureId) ?? null;
  }, [selectedFeatureId, data]);

  const voteMutation = useMutation({
    mutationFn: ({ featureId }: { featureId: number }) =>
      togglePortalVote(orgSlug!, repoName!, featureId, voterToken, ""),
    onMutate: async ({ featureId }) => {
      await queryClient.cancelQueries({ queryKey });
      const previousData = queryClient.getQueryData<PortalPage>(queryKey);

      queryClient.setQueryData<PortalPage>(queryKey, (old) => {
        if (!old) return old;
        const update = (f: PortalFeature): PortalFeature =>
          f.id !== featureId
            ? f
            : {
                ...f,
                has_voted: !f.has_voted,
                vote_count: f.has_voted ? f.vote_count - 1 : f.vote_count + 1,
              };
        return {
          ...old,
          requests: old.requests
            .map(update)
            .sort((a, b) => b.vote_count - a.vote_count),
          pending: old.pending
            .map(update)
            .sort((a, b) => b.vote_count - a.vote_count),
          in_progress: old.in_progress
            .map(update)
            .sort((a, b) => b.vote_count - a.vote_count),
          done: old.done
            .map(update)
            .sort((a, b) => b.vote_count - a.vote_count),
        };
      });

      return { previousData };
    },
    onError: (_err, _vars, context) => {
      if (context?.previousData) {
        queryClient.setQueryData(queryKey, context.previousData);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background">
        <PortalHeader orgSlug={orgSlug} repoOwner={null} repoName={repoName} />
        <main className="max-w-4xl mx-auto px-4 py-10 flex flex-col gap-12">
          <div className="flex flex-col gap-3">
            <Skeleton className="h-6 w-40" />
            <Skeleton className="h-[80px] w-full rounded-xl" />
            <Skeleton className="h-[80px] w-full rounded-xl" />
            <Skeleton className="h-[80px] w-full rounded-xl" />
          </div>
        </main>
      </div>
    );
  }

  if (isError) {
    console.log(error);

    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <p className="text-sm font-medium text-muted-foreground">
            Error loading portal
          </p>
        </div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <p className="text-sm font-medium text-muted-foreground">
            Portal not found
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            This portal is not public or does not exist.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <PortalHeader
        orgSlug={orgSlug}
        repoOwner={data.repo_owner}
        repoName={data.repo_name}
      />

      <main className="max-w-4xl mx-auto px-4 py-10 flex flex-col gap-12">
        <ProposalsBoard
          proposals={data.requests}
          onVote={(id) => voteMutation.mutate({ featureId: id })}
          onSelect={setSelectedFeatureId}
        />

        <RoadmapColumns
          pending={data.pending}
          inProgress={data.in_progress}
          done={data.done}
          onSelect={setSelectedFeatureId}
        />

        <RecentlyShipped
          features={data.done.sort((a, b) =>
            b.updated_at.localeCompare(a.updated_at),
          )}
          onSelect={setSelectedFeatureId}
        />
      </main>

      <FeatureSheet
        feature={selectedFeature}
        orgSlug={orgSlug!}
        repoName={repoName!}
        open={selectedFeatureId != null}
        onOpenChange={(open) => {
          if (!open) setSelectedFeatureId(null);
        }}
        onVote={(id) => voteMutation.mutate({ featureId: id })}
      />
    </div>
  );
}

function PortalHeader({
  orgSlug,
  repoOwner,
  repoName,
}: {
  orgSlug?: string;
  repoOwner: string | null;
  repoName?: string;
}) {
  return (
    <header className="border-b border-border bg-background/80 backdrop-blur-sm sticky top-0 z-10">
      <div className="max-w-4xl mx-auto px-4 h-14 flex items-center gap-3">
        <div className="flex items-center gap-2 text-sm font-medium">
          <GitFork className="size-4 text-muted-foreground" />
          {repoOwner ? (
            <span className="font-mono">
              <span className="text-muted-foreground font-normal">
                {repoOwner}/
              </span>
              {repoName}
            </span>
          ) : (
            <span className="font-mono text-muted-foreground">
              {orgSlug}/{repoName}
            </span>
          )}
        </div>
        <div className="ml-auto">
          <span className="text-xs text-muted-foreground">
            Community portal
          </span>
        </div>
      </div>
    </header>
  );
}
