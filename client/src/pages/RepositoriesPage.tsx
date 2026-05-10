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
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useOrgInstallationEvents } from "@/hooks/use-org-installation-events";
import { useOrgSetup } from "@/hooks/use-org-setup";
import {
  addRepository,
  formatApiError,
  getGithubAppInstallURL,
  listGithubAppInstallationRepos,
  listMyOrganizations,
  listOrgRepositories,
  type GitHubInstallationRepo,
  type Repository,
  getRepoMeta,
} from "@/lib/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ExternalLink,
  GitFork,
  Loader2Icon,
  Plus,
  SearchIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function repoKey(owner: string, name: string) {
  return `${owner.toLowerCase()}/${name.toLowerCase()}`;
}

export default function RepositoriesPage() {
  const queryClient = useQueryClient();
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  const { data: orgs = [], isLoading: orgsLoading } = useQuery({
    queryKey: ["organizations"],
    queryFn: () => listMyOrganizations(),
  });

  const org = orgs[0];
  const orgId = org?.id;

  useOrgInstallationEvents(orgId);

  const ghSetup = useOrgSetup(orgId);

  const { data: repos = [], isLoading: reposLoading } = useQuery({
    queryKey: ["organizations", orgId, "repositories"],
    queryFn: () => listOrgRepositories(orgId!),
    enabled: !!orgId,
  });

  const setupLoading = ghSetup.isLoading;
  const suspended = !setupLoading && ghSetup.isSuspended;
  const readyForRepos = !setupLoading && ghSetup.installed;

  async function openInstallPopup() {
    if (!orgId) return;
    try {
      const popup = window.open(
        "about:blank",
        "github_app_install",
        "popup,width=520,height=720",
      );
      const { install_url } = await getGithubAppInstallURL(orgId);
      if (popup) {
        popup.location.href = install_url;
      } else {
        window.location.href = install_url;
      }
    } catch (e) {
      toast.error(formatApiError(e));
    }
  }

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

  const addButton = (
    <Button
      variant="outline"
      size="sm"
      disabled={setupLoading || suspended || !orgId}
      className="shrink-0 mt-1"
      onClick={() => {
        if (setupLoading || suspended || !orgId) return;
        if (!readyForRepos) {
          toast.info("Connect GitHub first", {
            description:
              "Install the GitHub App from Settings, then add repositories here.",
            action: {
              label: "Settings",
              onClick: () => {
                window.location.href = "/settings";
              },
            },
          });
          return;
        }
        setAddDialogOpen(true);
      }}
    >
      <Plus data-icon="inline-start" />
      Add
    </Button>
  );

  const addControl =
    suspended || (!readyForRepos && !setupLoading && orgId) ? (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex">{addButton}</span>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          {suspended ? (
            <>
              The GitHub App is suspended. Unsuspend it on GitHub to add repos.
              {ghSetup.targetLogin ? (
                <>
                  {" "}
                  <a
                    href={
                      ghSetup.accountType === "Organization"
                        ? `https://github.com/organizations/${ghSetup.targetLogin}/settings/installations`
                        : "https://github.com/settings/installations"
                    }
                    target="_blank"
                    rel="noopener noreferrer"
                    className="underline underline-offset-2"
                  >
                    Manage on GitHub
                    <ExternalLink className="inline size-3 ml-0.5 align-text-bottom" />
                  </a>
                </>
              ) : null}
            </>
          ) : (
            <>
              Install the GitHub App to choose repos from your installation.{" "}
              <Link
                to="/settings"
                className="underline underline-offset-2 font-medium"
              >
                Open Settings
              </Link>{" "}
              or{" "}
              <button
                type="button"
                className="underline underline-offset-2 font-medium"
                onClick={(e) => {
                  e.preventDefault();
                  void openInstallPopup();
                }}
              >
                install now
              </button>
              .
            </>
          )}
        </TooltipContent>
      </Tooltip>
    ) : (
      addButton
    );

  return (
    <TooltipProvider>
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
          <div className="flex flex-col items-end gap-2 shrink-0">
            {addControl}
          </div>
        </div>

        {reposLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Skeleton className="h-[140px] w-full" />
            <Skeleton className="h-[140px] w-full" />
            <Skeleton className="h-[140px] w-full" />
            <Skeleton className="h-[140px] w-full" />
          </div>
        ) : repos.length === 0 ? (
          <div className="py-16 text-center rounded-lg bg-muted/50 flex flex-col items-center gap-3 px-4">
            <GitFork className="size-8 text-muted-foreground/40" />
            <p className="text-sm font-medium text-muted-foreground">
              No repositories yet
            </p>
            <p className="text-xs text-muted-foreground/80 max-w-md leading-relaxed">
              {ghSetup.isSuspended
                ? "Repositories require an active GitHub App installation. Unsuspend the app on GitHub, then add repos here."
                : ghSetup.installed
                  ? "Add a repository from your GitHub App installation to start tracking features and roadmap items."
                  : "Install the GitHub App on your GitHub account or organization, then add the repos you want to connect."}
            </p>
            {!ghSetup.installed &&
            !ghSetup.isSuspended &&
            !ghSetup.isLoading ? (
              <Button variant="outline" size="sm" asChild>
                <Link to="/settings">Go to Settings</Link>
              </Button>
            ) : null}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {repos.map((r) => (
              <RepoCard key={r.id} repo={r} />
            ))}
          </div>
        )}

        <AddRepositoriesDialog
          open={addDialogOpen}
          onOpenChange={setAddDialogOpen}
          orgId={orgId}
          linkedRepos={repos}
          onAdded={() => {
            queryClient.invalidateQueries({
              queryKey: ["organizations", orgId, "repositories"],
            });
            queryClient.invalidateQueries({
              queryKey: [
                "organizations",
                orgId,
                "github-installation-repositories",
              ],
            });
          }}
        />
      </div>
    </TooltipProvider>
  );
}

function AddRepositoriesDialog({
  open,
  onOpenChange,
  orgId,
  linkedRepos,
  onAdded,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  orgId: number | undefined;
  linkedRepos: Repository[];
  onAdded: () => void;
}) {
  const [search, setSearch] = useState("");
  const queryClient = useQueryClient();

  const linkedKeys = useMemo(() => {
    const set = new Set<string>();
    for (const r of linkedRepos) {
      set.add(repoKey(r.owner, r.name));
    }
    return set;
  }, [linkedRepos]);

  const {
    data: installationRepos = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["organizations", orgId, "github-installation-repositories"],
    queryFn: () => listGithubAppInstallationRepos(orgId!, 1),
    enabled: open && !!orgId,
  });

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    let list = installationRepos;
    if (q) {
      list = list.filter(
        (r) =>
          r.owner.toLowerCase().includes(q) ||
          r.name.toLowerCase().includes(q) ||
          r.full_name.toLowerCase().includes(q),
      );
    }
    return [...list].sort((a, b) =>
      `${a.owner}/${a.name}`.localeCompare(`${b.owner}/${b.name}`, undefined, {
        sensitivity: "base",
      }),
    );
  }, [installationRepos, search]);

  const addMutation = useMutation({
    mutationFn: ({ owner, name }: GitHubInstallationRepo) =>
      addRepository(orgId!, owner, name),
    onSuccess: () => {
      toast.success("Repository added");
      onAdded();
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "github-installation-repositories"],
      });
      onOpenChange(false);
      setSearch("");
    },
    onError: (err) => toast.error(formatApiError(err)),
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) setSearch("");
        onOpenChange(next);
      }}
    >
      <DialogContent
        className={cn(
          "gap-0 overflow-hidden p-0 sm:max-w-lg",
          "grid-rows-[auto_1fr]",
        )}
        showCloseButton
      >
        <DialogHeader className="p-4 pb-2 border-b border-border/60">
          <DialogTitle>Add repository</DialogTitle>
          <DialogDescription>
            Choose a repository from your GitHub App installation.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3 p-4 pt-3 min-h-0">
          <div className="relative">
            <SearchIcon className="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
            <Input
              placeholder="Filter by owner or name…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
              disabled={isLoading || !orgId}
            />
          </div>

          <div className="max-h-[min(360px,50vh)] overflow-y-auto rounded-lg border border-border/60 bg-muted/20">
            {isLoading ? (
              <div className="flex flex-col gap-2 p-3">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : isError ? (
              <div className="p-4 text-sm text-muted-foreground flex flex-col gap-2">
                <p>{formatApiError(error)}</p>
                <Button
                  variant="outline"
                  size="sm"
                  className="self-start"
                  onClick={() => void refetch()}
                >
                  Retry
                </Button>
              </div>
            ) : filtered.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground text-center">
                {installationRepos.length === 0
                  ? "No repositories returned for this installation. Adjust repo access on GitHub and try again."
                  : "No matches for your search."}
              </p>
            ) : (
              <ul className="flex flex-col divide-y divide-border/60">
                {filtered.map((r) => {
                  const key = repoKey(r.owner, r.name);
                  const already = linkedKeys.has(key);
                  const pending =
                    addMutation.isPending &&
                    addMutation.variables?.owner === r.owner &&
                    addMutation.variables?.name === r.name;
                  return (
                    <li key={key}>
                      <button
                        type="button"
                        disabled={already || pending || addMutation.isPending}
                        onClick={() => addMutation.mutate(r)}
                        className={cn(
                          "flex w-full items-center justify-between gap-3 px-3 py-2.5 text-left text-sm transition-colors",
                          "hover:bg-muted/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 ring-offset-background",
                          already &&
                            "opacity-50 cursor-not-allowed hover:bg-transparent",
                        )}
                      >
                        <span className="min-w-0 font-mono text-[13px] truncate">
                          <span className="text-muted-foreground">
                            {r.owner}/
                          </span>
                          {r.name}
                        </span>
                        <span className="shrink-0 flex items-center gap-2">
                          {pending ? (
                            <Loader2Icon className="size-4 animate-spin text-muted-foreground" />
                          ) : already ? (
                            <Badge color="zinc" className="text-[10px]">
                              Added
                            </Badge>
                          ) : (
                            <span className="text-xs text-muted-foreground">
                              Add
                            </span>
                          )}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
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
