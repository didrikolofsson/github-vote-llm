import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  getGitHubInstallUrl,
  getGitHubStatus,
  listMyOrganizations,
  listOrgMembers,
  listOrgRepositories,
} from "@/lib/api";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  ArrowRight,
  ArrowUpRight,
  Github,
  GitFork,
  Plus,
  Settings,
  UserPlus,
  Users,
} from "lucide-react";
import { Link, useNavigate } from "react-router-dom";

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function OrganizationDashboardPage() {
  const navigate = useNavigate();

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

  const { data: members = [], isLoading: membersLoading } = useQuery({
    queryKey: ["organizations", orgId, "members"],
    queryFn: () => listOrgMembers(orgId!),
    enabled: !!orgId,
  });

  const connectGitHub = useMutation({
    mutationFn: async () => {
      const { install_url } = await getGitHubInstallUrl();
      window.location.href = install_url;
    },
  });

  const recentRepos = repos.slice(0, 3);

  if (orgsLoading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-9 w-48" />
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Skeleton className="h-[120px] w-full" />
          <Skeleton className="h-[120px] w-full" />
          <Skeleton className="h-[120px] w-full" />
        </div>
        <Skeleton className="h-[200px] w-full" />
      </div>
    );
  }

  return (
    <div className="animate-slide-up flex flex-col gap-8 p-8 max-w-[1280px] mx-auto w-full">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Overview of {org?.name ?? "your organization"}
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {/* Repositories stat */}
        <Link to="/repositories" tabIndex={-1}>
          <Card className="group hover:bg-muted/30 transition-colors cursor-pointer h-full">
            <CardContent className="p-5">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center justify-center size-9 rounded-md bg-primary/8 text-primary">
                  <GitFork className="size-4" />
                </div>
                <ArrowUpRight className="size-4 text-muted-foreground/50 group-hover:text-muted-foreground transition-colors" />
              </div>
              {reposLoading ? (
                <Skeleton className="h-8 w-12 mb-1" />
              ) : (
                <div className="text-2xl font-semibold tabular-nums">{repos.length}</div>
              )}
              <p className="text-sm text-muted-foreground mt-0.5">Repositories</p>
            </CardContent>
          </Card>
        </Link>

        {/* Members stat */}
        <Link to="/settings" tabIndex={-1}>
          <Card className="group hover:bg-muted/30 transition-colors cursor-pointer h-full">
            <CardContent className="p-5">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center justify-center size-9 rounded-md bg-primary/8 text-primary">
                  <Users className="size-4" />
                </div>
                <ArrowUpRight className="size-4 text-muted-foreground/50 group-hover:text-muted-foreground transition-colors" />
              </div>
              {membersLoading ? (
                <Skeleton className="h-8 w-12 mb-1" />
              ) : (
                <div className="text-2xl font-semibold tabular-nums">{members.length}</div>
              )}
              <p className="text-sm text-muted-foreground mt-0.5">Members</p>
            </CardContent>
          </Card>
        </Link>

        {/* GitHub stat */}
        <Link to="/settings" tabIndex={-1}>
          <Card className="group hover:bg-muted/30 transition-colors cursor-pointer h-full">
            <CardContent className="p-5">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center justify-center size-9 rounded-md bg-primary/8 text-primary">
                  <Github className="size-4" />
                </div>
                <ArrowUpRight className="size-4 text-muted-foreground/50 group-hover:text-muted-foreground transition-colors" />
              </div>
              {ghStatusLoading ? (
                <>
                  <Skeleton className="h-6 w-24 mb-1" />
                  <Skeleton className="h-4 w-16 mt-1" />
                </>
              ) : ghStatus?.installed ? (
                <>
                  <Badge color="lime">Installed</Badge>
                  {ghStatus.login && (
                    <p className="text-sm text-muted-foreground mt-1.5">on @{ghStatus.login}</p>
                  )}
                </>
              ) : (
                <>
                  <Badge color="red">Not installed</Badge>
                  <p className="text-sm text-muted-foreground mt-1.5">GitHub App</p>
                </>
              )}
            </CardContent>
          </Card>
        </Link>
      </div>

      {/* Quick actions */}
      <div>
        <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-3">
          Quick actions
        </p>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link to="/repositories">
              <Plus />
              Add repository
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to="/settings">
              <UserPlus />
              Invite member
            </Link>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to="/settings">
              <Settings />
              Settings
            </Link>
          </Button>
          {!ghStatus?.installed && !ghStatusLoading && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => connectGitHub.mutate()}
              disabled={connectGitHub.isPending}
            >
              <Github />
              Install GitHub App
            </Button>
          )}
        </div>
      </div>

      {/* Recent repositories */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-[15px]">Recent repositories</CardTitle>
              <CardDescription className="mt-1">
                Your latest connected repositories
              </CardDescription>
            </div>
            <Button variant="ghost" size="sm" asChild>
              <Link to="/repositories">
                View all
                <ArrowRight />
              </Link>
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {reposLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : recentRepos.length === 0 ? (
            <div className="py-10 text-center rounded-lg bg-muted/50">
              <GitFork className="size-7 mx-auto text-muted-foreground/40 mb-3" />
              <p className="text-sm font-medium text-muted-foreground">
                No repositories yet
              </p>
              <p className="text-xs text-muted-foreground/70 mt-1 mb-4">
                Connect your first GitHub repository to get started.
              </p>
              <Button variant="outline" size="sm" onClick={() => navigate("/repositories")}>
                <Plus />
                Add repository
              </Button>
            </div>
          ) : (
            <ul className="flex flex-col gap-1">
              {recentRepos.map((r) => (
                <li key={r.id}>
                  <Link
                    to={`/repositories/${r.id}`}
                    className="flex items-center justify-between py-2 px-3 rounded-lg bg-muted/30 hover:bg-muted/60 transition-colors group"
                  >
                    <span className="text-sm font-mono">
                      {r.owner}/{r.name}
                    </span>
                    <div className="flex items-center gap-3 shrink-0 ml-4">
                      {r.created_at && (
                        <span className="text-xs text-muted-foreground">
                          {formatDate(r.created_at)}
                        </span>
                      )}
                      <ArrowRight className="size-3.5 text-muted-foreground/40 group-hover:text-muted-foreground transition-colors" />
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
