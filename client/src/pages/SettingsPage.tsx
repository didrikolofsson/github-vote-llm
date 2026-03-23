import { Alert, AlertDescription } from "@/components/ui/alert";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  formatApiError,
  getGitHubAuthorizeUrl,
  getGitHubStatus,
  listMyOrganizations,
  listOrgMembers,
  removeMember,
  updateMemberRole,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Github, MoreHorizontal, Users } from "lucide-react";
import { useEffect } from "react";

export default function SettingsPage() {
  const queryClient = useQueryClient();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const connected = params.get("github_connected");
    const error = params.get("github_error");
    if (connected === "1" || error) {
      queryClient.invalidateQueries({ queryKey: ["github-status"] });
      const url = new URL(window.location.href);
      url.searchParams.delete("github_connected");
      url.searchParams.delete("github_error");
      window.history.replaceState({}, "", url.pathname);
    }
  }, [queryClient]);

  return (
    <div className="animate-slide-up flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Manage your account and organization preferences
        </p>
      </div>

      <Tabs defaultValue="organization">
        <TabsList>
          <TabsTrigger value="organization">Organization</TabsTrigger>
          <TabsTrigger value="account">Account</TabsTrigger>
        </TabsList>

        <TabsContent value="organization" className="mt-6 flex flex-col gap-4">
          <OrganizationTab />
        </TabsContent>

        <TabsContent value="account" className="mt-6 flex flex-col gap-4">
          <AccountTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function OrganizationTab() {
  const queryClient = useQueryClient();

  const { data: orgs = [], isLoading: orgsLoading } = useQuery({
    queryKey: ["organizations"],
    queryFn: () => listMyOrganizations(),
  });

  const org = orgs[0];
  const orgId = org?.id;

  const { data: ghStatus, isLoading: ghLoading } = useQuery({
    queryKey: ["github-status"],
    queryFn: () => getGitHubStatus(),
    enabled: !!orgId,
  });

  const { data: members = [], isLoading: membersLoading } = useQuery({
    queryKey: ["organizations", orgId, "members"],
    queryFn: () => listOrgMembers(orgId!),
    enabled: !!orgId,
  });

  const connectGitHub = useMutation({
    mutationFn: async () => {
      const { authorize_url } = await getGitHubAuthorizeUrl();
      window.location.href = authorize_url;
    },
  });

  const removeMemberMutation = useMutation({
    mutationFn: (userId: number) => removeMember(orgId!, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "members"],
      });
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: number; role: "owner" | "member" }) =>
      updateMemberRole(orgId!, userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "members"],
      });
    },
  });

  if (orgsLoading) {
    return (
      <>
        <Skeleton className="h-[120px] w-full" />
        <Skeleton className="h-[200px] w-full" />
      </>
    );
  }

  return (
    <>
      {/* GitHub connection */}
      <Card>
        <CardHeader>
          <CardTitle className="text-[15px] flex items-center gap-2">
            <Github className="size-4" />
            GitHub connection
          </CardTitle>
          <CardDescription>
            Connect your GitHub account to enable repository management.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {ghLoading ? (
            <Skeleton className="h-4 w-48" />
          ) : ghStatus?.connected ? (
            <div className="flex items-center gap-2">
              <span className="size-2 rounded-full bg-green-500 shrink-0" />
              <span className="text-sm text-muted-foreground">
                Connected as{" "}
                <span className="font-medium text-foreground">
                  @{ghStatus.login}
                </span>
              </span>
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              <p className="text-sm text-muted-foreground">
                No GitHub account connected.
              </p>
              <div>
                <Button
                  onClick={() => connectGitHub.mutate()}
                  disabled={connectGitHub.isPending}
                >
                  <Github data-icon="inline-start" />
                  Connect GitHub
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Members */}
      <Card>
        <CardHeader>
          <CardTitle className="text-[15px] flex items-center gap-2">
            <Users className="size-4" />
            Members
          </CardTitle>
          <CardDescription>
            Manage who has access to this organization.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {membersLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : members.length === 0 ? (
            <div className="py-8 text-center rounded-lg bg-muted/50">
              <p className="text-sm text-muted-foreground">No members yet.</p>
            </div>
          ) : (
            <ul className="flex flex-col">
              {members.map((m, i) => (
                <li key={m.user_id}>
                  {i > 0 && <Separator className="my-1" />}
                  <div className="flex items-center justify-between py-2">
                    <div className="flex items-center gap-3">
                      <Avatar className="size-8 shrink-0">
                        <AvatarFallback className="text-xs">
                          {m.email.charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex flex-col">
                        <span className="text-sm text-foreground">{m.email}</span>
                        <Badge variant="secondary" className="w-fit text-[10px] px-1 py-0 h-4 mt-0.5">
                          {m.role}
                        </Badge>
                      </div>
                    </div>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8 text-muted-foreground"
                        >
                          <MoreHorizontal />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() =>
                            updateRoleMutation.mutate({
                              userId: m.user_id,
                              role: m.role === "member" ? "owner" : "member",
                            })
                          }
                          disabled={updateRoleMutation.isPending}
                        >
                          {m.role === "member" ? "Make owner" : "Make member"}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-destructive focus:text-destructive"
                          onClick={() => removeMemberMutation.mutate(m.user_id)}
                          disabled={removeMemberMutation.isPending}
                        >
                          Remove
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </>
  );
}

function AccountTab() {
  const { user, logout } = useAuth();

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-[15px]">Account</CardTitle>
          <CardDescription>Your personal account details.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3">
            <Avatar className="size-10 shrink-0">
              <AvatarFallback className="text-sm font-medium">
                {user?.email.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div>
              <p className="text-sm font-medium">{user?.email}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="border-destructive/40">
        <CardHeader>
          <CardTitle className="text-[15px]">Danger zone</CardTitle>
          <CardDescription>Irreversible account actions.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Sign out</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                End your current session.
              </p>
            </div>
            <Button variant="destructive" size="sm" onClick={() => void logout()}>
              Sign out
            </Button>
          </div>
        </CardContent>
      </Card>
    </>
  );
}
