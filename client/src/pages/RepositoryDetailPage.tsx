import { SetupGuard } from "@/components/setup/SetupGuard";
import { Badge, type BadgeColor } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RoadmapCanvas } from "@/components/roadmap/RoadmapCanvas";
import {
  getRepoMeta,
  getRoadmap,
  listRepositoryRuns,
  listMyOrganizations,
  listOrgRepositories,
  removeRepository,
  updateRepositoryPortalPublic,
} from "@/lib/api";
import type { Feature, Run, RunStatus } from "@/lib/api-schemas";
import { useOrgSetup } from "@/hooks/use-org-setup";
import { cn } from "@/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowUpRight,
  Bot,
  CalendarDays,
  Check,
  CircleDotDashed,
  Clock3,
  Copy,
  GitFork,
  Globe,
  ListChecks,
  type LucideIcon,
  Route,
  Sparkles,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

export default function RepositoryDetailPage() {
  const queryClient = useQueryClient();
  const { repoId } = useParams<{ repoId: string }>();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("details");
  const repoIdNum = repoId ? parseInt(repoId, 10) : undefined;

  const { data: orgs = [] } = useQuery({
    queryKey: ["organizations"],
    queryFn: () => listMyOrganizations(),
  });
  const org = orgs[0];
  const orgId = org?.id;

  const { data: repos = [] } = useQuery({
    queryKey: ["organizations", orgId, "repositories"],
    queryFn: () => listOrgRepositories(orgId!),
    enabled: !!orgId,
  });
  const repoData = repos.find((r) => r.id === repoIdNum);

  const ghSetup = useOrgSetup(orgId);

  const { data: repoMeta } = useQuery({
    queryKey: ["repository", repoIdNum, "meta"],
    queryFn: () => getRepoMeta(repoIdNum!),
    enabled: !!repoIdNum,
  });

  const { data: roadmap } = useQuery({
    queryKey: ["repositories", repoIdNum, "roadmap"],
    queryFn: () => getRoadmap(repoIdNum!),
    enabled: !!repoIdNum,
  });

  const { data: runs = [], isLoading: runsLoading } = useQuery({
    queryKey: ["repositories", repoIdNum, "runs"],
    queryFn: () => listRepositoryRuns(repoIdNum!),
    enabled: !!repoIdNum,
  });

  const removeRepo = useMutation({
    mutationFn: () => removeRepository(orgId!, repoIdNum!),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "repositories"],
      });
      navigate("/repositories");
    },
  });

  const togglePortal = useMutation({
    mutationFn: (portalPublic: boolean) =>
      updateRepositoryPortalPublic(repoIdNum!, portalPublic),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "repositories"],
      });
    },
  });

  const portalUrl =
    org?.slug && repoData
      ? `${window.location.origin}/portal/${org.slug}/${repoData.name}`
      : null;

  return (
    <div className="animate-slide-up flex flex-col">
      <Tabs value={activeTab} onValueChange={setActiveTab} className="gap-0">
        <div className="px-8 pt-8 w-full max-w-[1280px] mx-auto">
          <Link
            to="/repositories"
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
          >
            <ArrowLeft className="size-3.5" />
            Repositories
          </Link>
          <div className="relative overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
            <div className="absolute -right-16 -top-24 size-56 rounded-full bg-info/10 blur-3xl" />
            <div className="relative flex flex-col gap-4 p-4 lg:flex-row lg:items-start lg:justify-between">
              <div className="min-w-0">
                <div className="mb-2 flex flex-wrap items-center gap-2">
                  <Badge color="zinc" small className={setupBadgeClass(ghSetup)}>
                    {setupLabel(ghSetup)}
                  </Badge>
                  <Badge
                    color="zinc"
                    small
                    className={
                      repoData?.portal_public
                        ? "bg-info-muted text-info"
                        : undefined
                    }
                  >
                    {repoData?.portal_public
                      ? "Portal public"
                      : "Portal private"}
                  </Badge>
                </div>
                <h1 className="text-2xl font-semibold tracking-tight font-mono">
                  {repoData ? (
                    <>
                      <span className="text-muted-foreground font-normal">
                        {repoData.owner}/
                      </span>
                      {repoData.name}
                    </>
                  ) : (
                    <span className="text-muted-foreground">Loading…</span>
                  )}
                </h1>
                <p className="mt-1.5 max-w-xl text-sm leading-6 text-muted-foreground">
                  Plan features, publish signal, and launch implementation runs.
                </p>
                <div className="mt-4 flex flex-wrap gap-2">
                  {portalUrl && repoData?.portal_public ? (
                    <Button variant="outline" size="sm" asChild>
                      <a
                        href={portalUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Open portal
                        <ArrowUpRight data-icon="inline-end" />
                      </a>
                    </Button>
                  ) : null}
                  <Button size="sm" onClick={() => setActiveTab("roadmap")}>
                    Open roadmap
                  </Button>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-2 lg:w-[300px]">
                <RepoSignal
                  icon={Route}
                  label="Features"
                  value={repoMeta?.features ?? "—"}
                />
                <RepoSignal
                  icon={Bot}
                  label="Runs"
                  value={runs.length || repoMeta?.implementations || 0}
                />
                <RepoSignal
                  icon={Globe}
                  label="Portal"
                  value={repoData?.portal_public ? "Live" : "Draft"}
                />
              </div>
            </div>
          </div>
        </div>

        <div className="px-8 mt-8 w-full max-w-[1280px] mx-auto">
          <TabsList className="h-8">
            <TabsTrigger className="px-3" value="details">
              Details
            </TabsTrigger>
            <TabsTrigger className="px-3" value="roadmap">
              Roadmap
            </TabsTrigger>
            <TabsTrigger className="px-3" value="runs">
              Runs
            </TabsTrigger>
            <TabsTrigger className="px-3" value="settings">
              Settings
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent
          value="details"
          className="px-8 pb-8 mt-2 w-full max-w-[1280px] mx-auto flex flex-col gap-4"
        >
          <Card>
            <CardHeader>
              <CardTitle className="text-[15px]">Repository</CardTitle>
              <CardDescription>
                Overview and quick links for this repository.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              {repoData && (
                <div className="flex items-center justify-between py-3 border-b border-border/50">
                  <div className="flex items-center gap-2.5 text-sm text-muted-foreground">
                    <GitFork className="size-4 shrink-0" />
                    <span>GitHub</span>
                  </div>
                  <Button variant="outline" size="sm" asChild>
                    <a
                      href={`https://github.com/${repoData.owner}/${repoData.name}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {repoData.owner}/{repoData.name}
                      <ArrowUpRight className="size-3.5" />
                    </a>
                  </Button>
                </div>
              )}
              {repoData?.created_at && (
                <div className="flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2.5 text-muted-foreground">
                    <CalendarDays className="size-4 shrink-0" />
                    <span>Added</span>
                  </div>
                  <span className="text-muted-foreground">
                    {new Date(repoData.created_at).toLocaleDateString("en-US", {
                      month: "long",
                      day: "numeric",
                      year: "numeric",
                    })}
                  </span>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent
          value="roadmap"
          className="px-8 pb-8 mt-2 w-full max-w-[1280px] mx-auto"
        >
          <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
            <RoadmapOverview features={roadmap?.features ?? []} />
            <div style={{ height: "calc(100vh - 500px)", minHeight: 520 }}>
              {repoIdNum && <RoadmapCanvas repoId={repoIdNum} orgId={orgId} />}
            </div>
          </div>
        </TabsContent>

        <TabsContent
          value="runs"
          className="px-8 pb-8 mt-2 w-full max-w-[1280px] mx-auto"
        >
          <RunsLedger
            runs={runs}
            loading={runsLoading}
            features={roadmap?.features ?? []}
            orgId={orgId}
            onOpenRoadmap={() => setActiveTab("roadmap")}
          />
        </TabsContent>

        <TabsContent
          value="settings"
          className="px-8 pb-8 mt-2 w-full max-w-[1280px] mx-auto flex flex-col gap-4"
        >
          <PortalCard
            portalPublic={repoData?.portal_public ?? false}
            portalUrl={portalUrl}
            isPending={togglePortal.isPending || !repoIdNum}
            onToggle={(v) => togglePortal.mutate(v)}
          />
          <Card>
            <CardHeader>
              <CardTitle className="text-[15px]">Danger zone</CardTitle>
              <CardDescription>
                Remove this repository from your organization. Features and data
                will be preserved but the repository will no longer be managed
                here.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button
                variant="danger"
                size="sm"
                onClick={() => removeRepo.mutate()}
                disabled={removeRepo.isPending || !orgId || !repoIdNum}
              >
                <Trash2 />
                {removeRepo.isPending ? "Removing…" : "Remove repository"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

type SetupState = ReturnType<typeof useOrgSetup>;

function setupLabel(setup: SetupState) {
  if (setup.isLoading) return "Checking GitHub";
  if (setup.isSuspended) return "GitHub suspended";
  if (setup.installed) return "GitHub connected";
  return "GitHub setup needed";
}

function setupBadgeClass(setup: SetupState) {
  if (setup.isLoading) return undefined;
  if (setup.isSuspended) return "bg-warning-muted text-warning";
  if (setup.installed) return "bg-success-muted text-success";
  return "bg-warning-muted text-warning";
}

function RepoSignal({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: string | number;
}) {
  return (
    <div className="rounded-xl border border-border/70 bg-background p-2.5">
      <div className="mb-1.5 flex items-center justify-between text-muted-foreground">
        <Icon className="size-3.5" />
        <span className="text-[9px] uppercase tracking-[0.18em]">{label}</span>
      </div>
      <p className="font-mono text-lg font-semibold tracking-tight">{value}</p>
    </div>
  );
}

function RoadmapOverview({ features }: { features: Feature[] }) {
  const counts = features.reduce(
    (acc, feature) => {
      const status = feature.build_status ?? "pending";
      if (status === "in_progress") acc.inProgress += 1;
      else if (status === "done") acc.done += 1;
      else if (status === "stuck" || status === "rejected") acc.stuck += 1;
      else acc.pending += 1;
      return acc;
    },
    { pending: 0, inProgress: 0, done: 0, stuck: 0 },
  );

  const items = [
    {
      label: "Pending",
      value: counts.pending,
      className: "text-foreground",
    },
    {
      label: "In progress",
      value: counts.inProgress,
      className: "text-foreground",
    },
    {
      label: "Done",
      value: counts.done,
      className: "text-foreground",
    },
    {
      label: "Stuck",
      value: counts.stuck,
      className: "text-foreground",
    },
  ];

  return (
    <div className="relative border-b border-border/70 p-5">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,var(--muted)_0,transparent_34%)] opacity-80" />
      <div className="relative flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground">
            <Route className="size-3.5" />
            Roadmap
          </div>
          <h2 className="text-xl font-semibold tracking-tight">
            Strategic feature map
          </h2>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
            Drag features into sequence, connect dependencies, then launch runs
            from the feature drawer.
          </p>
        </div>
        <div className="grid grid-cols-4 gap-2">
          {items.map((item) => (
            <div
              key={item.label}
              className="min-w-20 rounded-xl border border-border/70 bg-background/80 p-2 text-center"
            >
              <p className={cn("font-mono text-lg font-semibold", item.className)}>
                {item.value}
              </p>
              <p className="text-[10px] text-muted-foreground">{item.label}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

const RUN_STATUS_META: Record<
  RunStatus,
  { label: string; color: BadgeColor; dot: string }
> = {
  pending: { label: "Pending", color: "amber", dot: "bg-amber-400" },
  running: { label: "Running", color: "cyan", dot: "bg-cyan-400" },
  completed: { label: "Completed", color: "lime", dot: "bg-lime-400" },
  failed: { label: "Failed", color: "red", dot: "bg-red-400" },
};

function formatRunDate(value: string | null) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function RunsLedger({
  runs,
  loading,
  features,
  orgId,
  onOpenRoadmap,
}: {
  runs: Run[];
  loading: boolean;
  features: Feature[];
  orgId: number | undefined;
  onOpenRoadmap: () => void;
}) {
  const featureById = new Map(features.map((feature) => [feature.id, feature]));
  const counts = runs.reduce<Record<RunStatus, number>>(
    (acc, run) => {
      acc[run.status] += 1;
      return acc;
    },
    { pending: 0, running: 0, completed: 0, failed: 0 },
  );

  if (loading) {
    return (
      <div className="grid gap-3">
        <Skeleton className="h-28 w-full rounded-2xl" />
        <Skeleton className="h-24 w-full rounded-xl" />
        <Skeleton className="h-24 w-full rounded-xl" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
        <div className="relative border-b border-border/70 p-5">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,var(--muted)_0,transparent_34%)] opacity-80" />
          <div className="relative flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div className="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground">
                <CircleDotDashed className="size-3.5" />
                Implementation runs
              </div>
              <h2 className="text-xl font-semibold tracking-tight">
                Execution ledger
              </h2>
              <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
                Every background implementation attempt for this repository,
                linked back to the feature that launched it.
              </p>
            </div>
            <div className="grid grid-cols-4 gap-2">
              {(Object.keys(RUN_STATUS_META) as RunStatus[]).map((status) => (
                <div
                  key={status}
                  className="min-w-20 rounded-xl border border-border/70 bg-background/80 p-2 text-center"
                >
                  <p className="font-mono text-lg font-semibold">
                    {counts[status]}
                  </p>
                  <p className="text-[10px] text-muted-foreground">
                    {RUN_STATUS_META[status].label}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </div>

        {runs.length === 0 ? (
          <div className="flex min-h-[300px] flex-col items-center justify-center gap-4 px-6 py-14 text-center">
            <div className="relative">
              <div className="absolute inset-0 rounded-full bg-info/20 blur-xl" />
              <div className="relative flex size-14 items-center justify-center rounded-2xl border border-border bg-background shadow-sm">
                <Sparkles className="size-6 text-muted-foreground" />
              </div>
            </div>
            <div>
              <p className="text-base font-semibold">No runs launched yet</p>
              <p className="mt-1 max-w-md text-sm leading-6 text-muted-foreground">
                Select a feature on the roadmap, review its implementation
                prompt, and start a run when the route is clear.
              </p>
            </div>
            <SetupGuard orgId={orgId}>
              <Button onClick={onOpenRoadmap}>
                Open roadmap
                <Route data-icon="inline-end" />
              </Button>
            </SetupGuard>
          </div>
        ) : (
          <div className="divide-y divide-border/70">
            {runs.map((run) => (
              <RunRow
                key={run.id}
                run={run}
                feature={featureById.get(run.feature_id)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function RunRow({ run, feature }: { run: Run; feature: Feature | undefined }) {
  const meta = RUN_STATUS_META[run.status];

  return (
    <div className="group grid gap-4 p-4 transition-colors hover:bg-muted/30 lg:grid-cols-[1fr_180px_160px] lg:items-center">
      <div className="min-w-0">
        <div className="mb-2 flex flex-wrap items-center gap-2">
          <Badge color={meta.color} small>
            <span className={cn("size-1.5 rounded-full", meta.dot)} />
            {meta.label}
          </Badge>
          <span className="font-mono text-[11px] text-muted-foreground">
            RUN-{String(run.id).padStart(4, "0")}
          </span>
        </div>
        <p className="truncate text-sm font-medium">
          {feature?.title ?? `Feature #${run.feature_id}`}
        </p>
        <p className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
          {run.prompt}
        </p>
      </div>
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Clock3 className="size-3.5" />
        <span>Started {formatRunDate(run.created_at)}</span>
      </div>
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <ListChecks className="size-3.5" />
        <span>
          {run.completed_at
            ? `Finished ${formatRunDate(run.completed_at)}`
            : run.status === "running"
              ? "Agent in progress"
              : "Awaiting result"}
        </span>
      </div>
    </div>
  );
}

function PortalCard({
  portalPublic,
  portalUrl,
  isPending,
  onToggle,
}: {
  portalPublic: boolean;
  portalUrl: string | null;
  isPending: boolean;
  onToggle: (v: boolean) => void;
}) {
  const [copied, setCopied] = useState(false);

  function copyUrl() {
    if (!portalUrl) return;
    navigator.clipboard.writeText(portalUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-[15px]">Community portal</CardTitle>
        <CardDescription>
          Publish a public page where users can vote on features and view the
          roadmap.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <Label
            htmlFor="portal-toggle"
            className="flex items-center gap-2 cursor-pointer"
          >
            <Globe className="size-4 text-muted-foreground" />
            <span className="text-sm">Public portal</span>
          </Label>
          <Switch
            id="portal-toggle"
            checked={portalPublic}
            onCheckedChange={onToggle}
            disabled={isPending}
          />
        </div>
        {portalPublic && portalUrl && (
          <div className="flex items-center gap-2 rounded-md border border-border bg-muted/50 px-3 py-2">
            <span className="text-xs text-muted-foreground font-mono flex-1 truncate">
              {portalUrl}
            </span>
            <Button
              variant="ghost"
              size="icon"
              className="size-6 shrink-0"
              onClick={copyUrl}
            >
              {copied ? (
                <Check className="size-3.5 text-lime-400" />
              ) : (
                <Copy className="size-3.5" />
              )}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="size-6 shrink-0"
              asChild
            >
              <a href={portalUrl} target="_blank" rel="noopener noreferrer">
                <ArrowUpRight className="size-3.5" />
              </a>
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
