import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  getRepoMeta,
  listMyOrganizations,
  listOrgRepositories,
  type Repository,
} from "@/lib/api";
import { useQuery } from "@tanstack/react-query";
import { GitFork, Plus } from "lucide-react";
import { Link } from "react-router-dom";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function RepositoriesPage() {
  const { data: orgs = [], isLoading: orgsLoading } = useQuery({
    queryKey: ["organizations"],
    queryFn: () => listMyOrganizations(),
  });

  const org = orgs[0];
  const orgId = org?.id;

  const { data: repos = [], isLoading: reposLoading } = useQuery({
    queryKey: ["organizations", orgId, "repositories"],
    queryFn: () => listOrgRepositories(orgId!),
    enabled: !!orgId,
  });

  if (orgsLoading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-9 w-48" />
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
        </div>
      </div>
    );
  }

  return (
    <div className="animate-slide-up flex flex-col gap-6 p-8 max-w-[1280px] mx-auto w-full">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">
            Repositories
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Manage repositories connected to your organization
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled
          className="shrink-0 mt-1"
        >
          <Plus data-icon="inline-start" />
          Add
        </Button>
      </div>

      {reposLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
        </div>
      ) : repos.length === 0 ? (
        <div className="py-16 text-center rounded-lg bg-muted/50">
          <GitFork className="size-8 mx-auto text-muted-foreground/40 mb-3" />
          <p className="text-sm font-medium text-muted-foreground">
            No repositories yet
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            Repository selection will be enabled once the GitHub backend is
            implemented.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {repos.map((r) => (
            <RepoCard key={r.id} repo={r} />
          ))}
        </div>
      )}

    </div>
  );
}

function RepoCard({ repo }: { repo: Repository }) {
  const { data: meta, isLoading: metaLoading } = useQuery({
    queryKey: ["repository", repo.id, "meta"],
    queryFn: () => getRepoMeta(repo.id),
  });
  return (
    <Link to={`/repositories/${repo.id}`} className="group block">
      <Card className="h-full transition-all duration-150 group-hover:shadow">
        <CardContent className="p-5 flex flex-col gap-3 h-full">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <p className="text-xs text-muted-foreground font-mono">
                {repo.owner}/
              </p>
              <p className="text-[15px] font-semibold font-mono leading-tight truncate">
                {repo.name}
              </p>
            </div>
            {metaLoading ? (
              <Skeleton className="h-5 w-[72px] shrink-0 mt-0.5 rounded-full" />
            ) : (
              <Badge
                color={meta?.status === "active" ? "lime" : "zinc"}
                className="shrink-0 mt-0.5"
              >
                {meta?.status ?? "—"}
              </Badge>
            )}
          </div>

          {metaLoading ? (
            <div className="flex flex-col gap-2 flex-1">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-[88%]" />
            </div>
          ) : (
            <p className="text-sm text-muted-foreground line-clamp-2 flex-1">
              {meta?.description ?? "—"}
            </p>
          )}

          <div className="flex items-center justify-between pt-2 border-t border-border/50 text-xs text-muted-foreground">
            {metaLoading ? (
              <div className="flex items-center gap-3">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-3 w-28" />
              </div>
            ) : (
              <div className="flex items-center gap-3">
                <span>{meta?.features ?? "—"} features</span>
                <span>{meta?.implementations ?? "—"} implementations</span>
              </div>
            )}
            {repo.created_at && (
              <span className="shrink-0">
                Added {formatDate(repo.created_at)}
              </span>
            )}
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
