import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
  getGitHubStatus,
  listAvailableRepositories,
  listMyOrganizations,
  listOrgRepositories,
  removeRepository,
  type OrgRepository,
} from "@/lib/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, LayoutGrid, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";

export default function OrganizationDashboardPage() {
  const queryClient = useQueryClient();
  const [addRepoOpen, setAddRepoOpen] = useState(false);

  const { data: orgs = [], isLoading: orgsLoading } = useQuery({
    queryKey: ["organizations"],
    queryFn: () => listMyOrganizations(),
  });

  const org = orgs[0];
  const orgId = org?.id;

  const { data: ghStatus, isLoading: ghStatusLoading } = useQuery({
    queryKey: ["github-status"],
    queryFn: () => getGitHubStatus(),
    enabled: !!orgId,
  });

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

  const removeRepo = useMutation({
    mutationFn: ({ owner, repo }: { owner: string; repo: string }) =>
      removeRepository(orgId!, owner, repo),
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
        <Skeleton className="h-[300px] w-full" />
      </div>
    );
  }

  return (
    <div className="animate-slide-up flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Your connected repositories
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-[15px] flex items-center gap-2">
                Repositories
              </CardTitle>
              <CardDescription className="mt-1">
                Repositories connected to this organization.
              </CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setAddRepoOpen(true)}
              disabled={!ghStatus?.connected}
            >
              <Plus data-icon="inline-start" />
              Add
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {reposLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : repos.length === 0 ? (
            <div className="py-12 text-center rounded-lg bg-muted/50">
              <LayoutGrid className="size-8 mx-auto text-muted-foreground/40 mb-3" />
              <p className="text-sm font-medium text-muted-foreground">
                No repositories yet
              </p>
              <p className="text-xs text-muted-foreground/70 mt-1">
                {ghStatus?.connected ? (
                  "Click Add to connect your first repository."
                ) : (
                  <>
                    Connect GitHub in{" "}
                    <Link to="/settings" className="underline underline-offset-2">
                      Settings
                    </Link>{" "}
                    to add repositories.
                  </>
                )}
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-3">
            {!ghStatus?.connected && !ghStatusLoading && (
              <div className="flex items-start gap-2.5 rounded-lg border border-amber-200 bg-amber-50 px-3.5 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-400">
                <AlertTriangle className="mt-0.5 size-4 shrink-0" />
                <span>
                  GitHub account disconnected. Repositories are preserved but you can't add new ones until you{" "}
                  <Link to="/settings" className="font-medium underline underline-offset-2">
                    reconnect
                  </Link>
                  .
                </span>
              </div>
            )}
            <ul className="flex flex-col gap-2">
              {repos.map((r) => (
                <li
                  key={`${r.owner}/${r.repo}`}
                  className="flex items-center justify-between py-2 px-3 rounded-lg bg-muted/30"
                >
                  <span className="text-sm font-mono">
                    {r.owner}/{r.repo}
                  </span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-8 text-muted-foreground hover:text-destructive"
                    onClick={() =>
                      removeRepo.mutate({ owner: r.owner, repo: r.repo })
                    }
                    disabled={removeRepo.isPending}
                  >
                    <Trash2 />
                  </Button>
                </li>
              ))}
            </ul>
            </div>
          )}
        </CardContent>
      </Card>

      <AddRepoDialog
        open={addRepoOpen}
        onOpenChange={setAddRepoOpen}
        orgId={orgId}
        existingRepos={repos ?? []}
        onAdd={addRepo.mutate}
        adding={addRepo.isPending}
      />
    </div>
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
  existingRepos: OrgRepository[];
  onAdd: (params: { owner: string; repo: string }) => void;
  adding: boolean;
}) {
  const [page, setPage] = useState(1);
  const existingSet = new Set(existingRepos.map((r) => `${r.owner}/${r.repo}`));

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
