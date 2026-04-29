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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { RoadmapCanvas } from "@/components/roadmap/RoadmapCanvas";
import {
  createRun as createFeatureRun,
  listMyOrganizations,
  listOrgRepositories,
  removeRepository,
  updateRepositoryPortalPublic,
} from "@/lib/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowUpRight,
  CalendarDays,
  Check,
  Copy,
  GitFork,
  Globe,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

export default function RepositoryDetailPage() {
  const queryClient = useQueryClient();
  const { repoId } = useParams<{ repoId: string }>();
  const navigate = useNavigate();
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

  const removeRepo = useMutation({
    mutationFn: () => removeRepository(orgId!, repoIdNum!),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "repositories"],
      });
      navigate("/repositories");
    },
  });

  type CreateFeatureRunParams = {
    prompt: string;
    featureId: number;
    createdByUserId: number;
  };
  const createRun = useMutation({
    mutationFn: ({
      prompt,
      featureId,
      createdByUserId,
    }: CreateFeatureRunParams) =>
      createFeatureRun(prompt, featureId, createdByUserId),
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
      <div className="px-8 pt-8 w-full max-w-[1280px] mx-auto">
        <Link
          to="/repositories"
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
        >
          <ArrowLeft className="size-3.5" />
          Repositories
        </Link>
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
      </div>

      <Tabs defaultValue="details">
        <div className="px-8 mt-6 w-full max-w-[1280px] mx-auto">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="roadmap">Roadmap</TabsTrigger>
            <TabsTrigger value="runs">Runs</TabsTrigger>
            <TabsTrigger value="settings">Settings</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent
          value="details"
          className="px-8 pb-8 mt-6 w-full max-w-[1280px] mx-auto flex flex-col gap-4"
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

        <TabsContent value="roadmap" className="mt-4">
          <div style={{ height: "calc(100vh - 240px)" }}>
            {repoIdNum && <RoadmapCanvas repoId={repoIdNum} />}
          </div>
        </TabsContent>

        <TabsContent
          value="runs"
          className="px-8 pb-8 mt-6 w-full max-w-[1280px] mx-auto"
        >
          <Button
            onClick={() =>
              createRun.mutate({
                prompt: "Create a new implementation for the feature",
                featureId: 1,
                createdByUserId: 1,
              })
            }
          >
            Create run
          </Button>
          <div className="py-16 text-center rounded-lg bg-muted/50">
            <p className="text-sm font-medium text-muted-foreground">
              No implementations yet
            </p>
            <p className="text-xs text-muted-foreground/70 mt-1">
              AI-driven implementations and their pull requests will appear
              here.
            </p>
          </div>
        </TabsContent>

        <TabsContent
          value="settings"
          className="px-8 pb-8 mt-6 w-full max-w-[1280px] mx-auto flex flex-col gap-4"
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
