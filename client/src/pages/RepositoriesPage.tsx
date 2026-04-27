import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  addRepository,
  formatApiError,
  getRepoMeta,
  listAvailableRepositories,
  listMyOrganizations,
  listOrgRepositories,
  type Repository,
} from "@/lib/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAccount } from "@/lib/account";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { AlertTriangle, GitFork, Plus, Zap } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function RepositoriesPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [addRepoOpen, setAddRepoOpen] = useState(false);
  const { status } = useAccount();
  const isActivated = status === "active";

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

  const addRepo = useMutation({
    mutationFn: ({ owner, repo }: { owner: string; repo: string }) =>
      addRepository(orgId!, owner, repo),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "repositories"],
      });
    },
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
        {isActivated ? (
          <Button
            variant="outline"
            size="sm"
            onClick={() => setAddRepoOpen(true)}
            className="shrink-0 mt-1"
          >
            <Plus data-icon="inline-start" />
            Add
          </Button>
        ) : (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="shrink-0 mt-1">
                <Button variant="outline" size="sm" disabled>
                  <Plus data-icon="inline-start" />
                  Add
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>
              {status === "inactive"
                ? "Connect your GitHub account first"
                : "Install the GitHub App to add repositories"}
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      {!isActivated && repos.length > 0 && (
        <div className="flex items-start gap-2.5 rounded-lg border border-amber-200 bg-amber-50 px-3.5 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-400">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" />
          <span>
            GitHub App not installed. Repositories are preserved but you
            can't add new ones until you{" "}
            <Link
              to={status === "inactive" ? "/setup/connect-github" : "/setup/install-app"}
              className="font-medium underline underline-offset-2"
            >
              complete setup
            </Link>
            .
          </span>
        </div>
      )}

      {reposLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
          <Skeleton className="h-[140px] w-full" />
        </div>
      ) : repos.length === 0 ? (
        !isActivated ? (
          <div className="py-16 text-center rounded-lg bg-muted/50 flex flex-col items-center">
            <Zap className="size-7 mx-auto text-muted-foreground/40 mb-3" />
            <p className="text-sm font-medium text-muted-foreground">
              Activate your organization to add repositories
            </p>
            <p className="text-xs text-muted-foreground/70 mt-1 mb-4">
              {status === "inactive"
                ? "Start by connecting your GitHub account."
                : "Install the GitHub App to connect your first repository."}
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                navigate(
                  status === "inactive"
                    ? "/setup/connect-github"
                    : "/setup/install-app",
                )
              }
            >
              {status === "inactive" ? "Connect GitHub" : "Install GitHub App"}
            </Button>
          </div>
        ) : (
        <div className="py-16 text-center rounded-lg bg-muted/50">
          <GitFork className="size-8 mx-auto text-muted-foreground/40 mb-3" />
          <p className="text-sm font-medium text-muted-foreground">
            No repositories yet
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            Click Add to connect your first repository.
          </p>
        </div>
        )
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {repos.map((r) => (
            <RepoCard key={r.id} repo={r} />
          ))}
        </div>
      )}

      <AddRepoDialog
        open={addRepoOpen}
        onOpenChange={setAddRepoOpen}
        orgId={orgId}
        existingRepos={repos}
        onAdd={addRepo.mutate}
        adding={addRepo.isPending}
      />
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

function AddRepoDialog({
  open,
  onOpenChange,
  orgId,
  existingRepos,
  onAdd,
  adding,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgId?: number;
  existingRepos: Repository[];
  onAdd: (params: { owner: string; repo: string }) => void;
  adding: boolean;
}) {
  const [page, setPage] = useState(1);
  const existingSet = new Set(existingRepos.map((r) => `${r.owner}/${r.name}`));

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["available-repos", orgId, page],
    queryFn: () => listAvailableRepositories(page),
    enabled: open && !!orgId,
  });

  const repos = data?.repositories ?? [];
  const hasMore = data?.has_more ?? false;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Add repository</DialogTitle>
          <DialogDescription>
            Select a repository from your GitHub account to add to this
            organization.
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-[320px] overflow-y-auto mt-4">
          {isError && error && (
            <Alert variant="destructive" className="mb-4">
              <AlertDescription>{formatApiError(error)}</AlertDescription>
            </Alert>
          )}
          {isLoading ? (
            <div className="flex flex-col gap-2 py-4">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : repos.length === 0 ? (
            isError ? null : (
              <p className="text-sm text-muted-foreground py-8 text-center">
                No repositories found.
              </p>
            )
          ) : (
            <ul className="flex flex-col gap-1">
              {repos.map((r) => {
                const key = `${r.owner}/${r.repo}`;
                const alreadyAdded = existingSet.has(key);
                return (
                  <li key={key}>
                    <Button
                      variant="ghost"
                      className="w-full justify-between"
                      onClick={() =>
                        !alreadyAdded && onAdd({ owner: r.owner, repo: r.repo })
                      }
                      disabled={alreadyAdded || adding}
                    >
                      <span className="font-mono">{key}</span>
                      {alreadyAdded ? (
                        <span className="text-xs text-muted-foreground">
                          Added
                        </span>
                      ) : (
                        <Plus data-icon="inline-end" />
                      )}
                    </Button>
                  </li>
                );
              })}
            </ul>
          )}
          {hasMore && (
            <div className="mt-4 flex justify-center">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => p + 1)}
              >
                Load more
              </Button>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
