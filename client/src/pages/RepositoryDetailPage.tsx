import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { listMyOrganizations, listOrgRepositories, removeRepository } from "@/lib/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, ArrowUpRight, CalendarDays, GitFork, Trash2 } from "lucide-react";
import { Link, useNavigate, useParams } from "react-router-dom";

export default function RepositoryDetailPage() {
  const { owner, repo } = useParams<{ owner: string; repo: string }>();
  const navigate = useNavigate();

  const { data: orgs = [] } = useQuery({
    queryKey: ["organizations"],
    queryFn: () => listMyOrganizations(),
  });
  const orgId = orgs[0]?.id;

  const { data: repos = [] } = useQuery({
    queryKey: ["organizations", orgId, "repositories"],
    queryFn: () => listOrgRepositories(orgId!),
    enabled: !!orgId,
  });
  const repoData = repos.find((r) => r.owner === owner && r.repo === repo);

  const queryClient = useQueryClient();
  const removeRepo = useMutation({
    mutationFn: () => removeRepository(orgId!, owner!, repo!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["organizations", orgId, "repositories"] });
      navigate("/repositories");
    },
  });

  return (
    <div className="animate-slide-up flex flex-col gap-6">
      <div>
        <Link
          to="/repositories"
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"
        >
          <ArrowLeft className="size-3.5" />
          Repositories
        </Link>
        <h1 className="text-2xl font-semibold tracking-tight font-mono">
          <span className="text-muted-foreground font-normal">{owner}/</span>
          {repo}
        </h1>
      </div>

      <Tabs defaultValue="details">
        <TabsList>
          <TabsTrigger value="details">Details</TabsTrigger>
          <TabsTrigger value="roadmap">Roadmap</TabsTrigger>
          <TabsTrigger value="runs">Runs</TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
        </TabsList>

        <TabsContent value="details" className="mt-6 flex flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-[15px]">Repository</CardTitle>
              <CardDescription>Overview and quick links for this repository.</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <div className="flex items-center justify-between py-3 border-b border-border/50">
                <div className="flex items-center gap-2.5 text-sm text-muted-foreground">
                  <GitFork className="size-4 shrink-0" />
                  <span>GitHub</span>
                </div>
                <Button variant="outline" size="sm" asChild>
                  <a
                    href={`https://github.com/${owner}/${repo}`}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {owner}/{repo}
                    <ArrowUpRight className="size-3.5" />
                  </a>
                </Button>
              </div>
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

        <TabsContent value="roadmap" className="mt-6">
          <div className="py-16 text-center rounded-lg bg-muted/50">
            <p className="text-sm font-medium text-muted-foreground">Roadmap coming soon</p>
            <p className="text-xs text-muted-foreground/70 mt-1">
              Approved proposals and their status will appear here.
            </p>
          </div>
        </TabsContent>

        <TabsContent value="runs" className="mt-6">
          <div className="py-16 text-center rounded-lg bg-muted/50">
            <p className="text-sm font-medium text-muted-foreground">No implementations yet</p>
            <p className="text-xs text-muted-foreground/70 mt-1">
              AI-driven implementations and their pull requests will appear here.
            </p>
          </div>
        </TabsContent>

        <TabsContent value="settings" className="mt-6 flex flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-[15px]">Danger zone</CardTitle>
              <CardDescription>
                Remove this repository from your organization. Proposals and data will be
                preserved but the repository will no longer be managed here.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => removeRepo.mutate()}
                disabled={removeRepo.isPending || !orgId}
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
