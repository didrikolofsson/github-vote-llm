import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  deleteUser,
  disconnectGitHub,
  formatApiError,
  getGitHubInstallUrl,
  getGitHubStatus,
  getMe,
  listMyOrganizations,
  listOrgMembers,
  removeMember,
  updateMemberRole,
  updateOrganization,
  updateOrganizationSlug,
  updateUsername,
} from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Github, MoreHorizontal } from "lucide-react";
import { useEffect, useState } from "react";
import { userRoleToBadgeColor } from "@/lib/utils";

export default function SettingsPage() {
  const queryClient = useQueryClient();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const installed = params.get("github_installed");
    const error = params.get("github_error");
    if (installed === "1" || error) {
      queryClient.invalidateQueries({ queryKey: ["github-status"] });
      const url = new URL(window.location.href);
      url.searchParams.delete("github_installed");
      url.searchParams.delete("github_error");
      window.history.replaceState({}, "", url.pathname);
    }
  }, [queryClient]);

  return (
    <div className="animate-slide-up flex flex-col gap-6 p-8 max-w-[1280px] mx-auto w-full">
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

  const [orgName, setOrgName] = useState(org?.name ?? "");
  const [orgSlug, setOrgSlug] = useState(org?.slug ?? "");
  const [saveError, setSaveError] = useState<string | null>(null);
  const [slugError, setSlugError] = useState<string | null>(null);

  useEffect(() => {
    if (org?.name) setOrgName(org.name);
    if (org?.slug) setOrgSlug(org.slug);
  }, [org?.name, org?.slug]);

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
      const { install_url } = await getGitHubInstallUrl();
      window.location.href = install_url;
    },
  });

  const disconnectGitHubMutation = useMutation({
    mutationFn: () => disconnectGitHub(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["github-status"] });
    },
  });

  const updateOrgMutation = useMutation({
    mutationFn: () => updateOrganization(orgId!, orgName.trim()),
    onSuccess: () => {
      setSaveError(null);
      queryClient.invalidateQueries({ queryKey: ["organizations"] });
    },
    onError: (err) => setSaveError(formatApiError(err)),
  });

  const updateSlugMutation = useMutation({
    mutationFn: () => updateOrganizationSlug(orgId!, orgSlug.trim()),
    onSuccess: () => {
      setSlugError(null);
      queryClient.invalidateQueries({ queryKey: ["organizations"] });
    },
    onError: (err) => setSlugError(formatApiError(err)),
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
    mutationFn: ({
      userId,
      role,
    }: {
      userId: number;
      role: "owner" | "member";
    }) => updateMemberRole(orgId!, userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["organizations", orgId, "members"],
      });
    },
  });

  if (orgsLoading) {
    return (
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-4">
        <Skeleton className="h-[200px] w-full" />
        <div className="flex flex-col gap-4">
          <Skeleton className="h-[100px] w-full" />
          <Skeleton className="h-[200px] w-full" />
        </div>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-4 items-start">
      {/* Left: org info form */}
      <Card>
        <CardHeader>
          <CardTitle className="text-[15px] flex items-center gap-2">
            Organization
          </CardTitle>
          <CardDescription>Edit your organization details.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="org-name">Name</Label>
            <Input
              id="org-name"
              value={orgName}
              onChange={(e) => setOrgName(e.target.value)}
              placeholder="Organization name"
            />
          </div>
          {saveError && <p className="text-sm text-destructive">{saveError}</p>}
          <div>
            <Button
              onClick={() => updateOrgMutation.mutate()}
              disabled={
                updateOrgMutation.isPending ||
                !orgName.trim() ||
                orgName.trim() === org?.name
              }
              size="sm"
            >
              {updateOrgMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </div>

          <Separator />

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="org-slug">
              Portal URL slug
              <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                Used in your community portal URL
              </span>
            </Label>
            <Input
              id="org-slug"
              value={orgSlug}
              onChange={(e) => setOrgSlug(e.target.value)}
              placeholder="my-organization"
              className="font-mono text-sm"
            />
            {org?.slug && (
              <p className="text-xs text-muted-foreground">
                Portal URL:{" "}
                <span className="font-mono">
                  {window.location.origin}/portal/{orgSlug}/…
                </span>
              </p>
            )}
          </div>
          {slugError && <p className="text-sm text-destructive">{slugError}</p>}
          <div>
            <Button
              onClick={() => updateSlugMutation.mutate()}
              disabled={
                updateSlugMutation.isPending ||
                !orgSlug.trim() ||
                orgSlug.trim() === org?.slug
              }
              size="sm"
              variant="outline"
            >
              {updateSlugMutation.isPending ? "Saving..." : "Update slug"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Right: GitHub + Members */}
      <div className="flex flex-col gap-4">
        {/* GitHub App installation */}
        <Card>
          <CardHeader>
            <CardTitle className="text-[15px] flex items-center gap-2">
              GitHub App
            </CardTitle>
            <CardDescription>
              Install the GitHub App to grant repository access for automated
              PRs.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {ghLoading ? (
              <Skeleton className="h-4 w-48" />
            ) : ghStatus?.installed ? (
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span
                    className={`size-2 rounded-full shrink-0 ${
                      ghStatus.suspended ? "bg-amber-400" : "bg-lime-400"
                    }`}
                  />
                  <span className="text-sm text-muted-foreground">
                    Installed on{" "}
                    <span className="font-medium text-foreground">
                      @{ghStatus.login}
                    </span>
                    {ghStatus.repository_selection && (
                      <span className="ml-1.5 text-xs">
                        ({ghStatus.repository_selection === "all"
                          ? "all repos"
                          : "selected repos"})
                      </span>
                    )}
                    {ghStatus.suspended && (
                      <span className="ml-1.5 text-xs text-amber-600">
                        suspended
                      </span>
                    )}
                  </span>
                </div>
                <Button
                  variant="outline"
                  size="xs"
                  onClick={() => disconnectGitHubMutation.mutate()}
                  disabled={disconnectGitHubMutation.isPending}
                >
                  {disconnectGitHubMutation.isPending
                    ? "Removing..."
                    : "Disconnect"}
                </Button>
              </div>
            ) : (
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="size-2 rounded-full bg-muted-foreground/40 shrink-0" />
                  <span className="text-sm text-muted-foreground">
                    Not installed
                  </span>
                </div>
                <Button
                  size="xs"
                  onClick={() => connectGitHub.mutate()}
                  disabled={connectGitHub.isPending}
                >
                  <Github data-icon="inline-start" />
                  {connectGitHub.isPending ? "Redirecting..." : "Install"}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Members */}
        <Card>
          <CardHeader>
            <CardTitle className="text-[15px] flex items-center gap-2">
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
                          <span className="text-sm text-foreground">
                            {m.email}
                          </span>
                          <Badge color={userRoleToBadgeColor(m.role)} small>
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
                            onClick={() =>
                              removeMemberMutation.mutate(m.user_id)
                            }
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
      </div>
    </div>
  );
}

function AccountTab() {
  const { user, logout } = useAuth();
  const queryClient = useQueryClient();
  const [usernameInput, setUsernameInput] = useState("");
  const [usernameError, setUsernameError] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteConfirmEmail, setDeleteConfirmEmail] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const { data: profile, isLoading: profileLoading } = useQuery({
    queryKey: ["users", "me"],
    queryFn: () => getMe(),
  });

  useEffect(() => {
    if (profile?.username) setUsernameInput(profile.username);
  }, [profile?.username]);

  const updateUsernameMutation = useMutation({
    mutationFn: () => updateUsername(usernameInput.trim()),
    onSuccess: () => {
      setUsernameError(null);
      queryClient.invalidateQueries({ queryKey: ["users", "me"] });
    },
    onError: (err) => setUsernameError(formatApiError(err)),
  });

  const deleteAccountMutation = useMutation({
    mutationFn: () => {
      const id = profile?.id ?? user?.id;
      if (id == null) throw new Error("Account not loaded");
      return deleteUser(id);
    },
    onSuccess: async () => {
      setDeleteDialogOpen(false);
      setDeleteConfirmEmail("");
      setDeleteError(null);
      queryClient.clear();
      await logout();
    },
    onError: (err) => setDeleteError(formatApiError(err)),
  });

  const displayName = profile?.username ?? user?.email ?? "";
  const emailMatchesConfirm =
    !!user?.email &&
    deleteConfirmEmail.trim().toLowerCase() === user.email.toLowerCase();

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-[15px]">Account</CardTitle>
          <CardDescription>Your personal account details.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex items-center gap-3">
            <Avatar className="size-10 shrink-0">
              <AvatarFallback className="text-sm font-medium">
                {displayName.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div>
              <p className="text-sm font-medium">
                {profile?.username ?? user?.email}
              </p>
              {profile?.username && (
                <p className="text-xs text-muted-foreground">{user?.email}</p>
              )}
            </div>
          </div>

          <Separator />

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="username">Username</Label>
            {profileLoading ? (
              <Skeleton className="h-9 w-full" />
            ) : (
              <Input
                id="username"
                value={usernameInput}
                onChange={(e) => setUsernameInput(e.target.value)}
                placeholder={user?.email}
              />
            )}
            <p className="text-xs text-muted-foreground">
              Used as your display name. Defaults to your email if not set.
            </p>
          </div>
          {usernameError && (
            <p className="text-sm text-destructive">{usernameError}</p>
          )}
          <div>
            <Button
              size="sm"
              onClick={() => updateUsernameMutation.mutate()}
              disabled={
                updateUsernameMutation.isPending ||
                !usernameInput.trim() ||
                usernameInput.trim() === (profile?.username ?? "")
              }
            >
              {updateUsernameMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card className="border-destructive/40">
        <CardHeader>
          <CardTitle className="text-[15px]">Danger zone</CardTitle>
          <CardDescription>Irreversible account actions.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium">Delete account</p>
              <p className="text-xs text-muted-foreground mt-0.5 max-w-md">
                Permanently remove your user, sessions, and GitHub connection.
                If you are the only member of an organization, that organization
                and its linked repositories are removed. If others remain,
                ownership transfers to another member.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              className="shrink-0"
              onClick={() => {
                setDeleteConfirmEmail("");
                setDeleteError(null);
                setDeleteDialogOpen(true);
              }}
            >
              Delete account
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={deleteDialogOpen}
        onOpenChange={(open) => {
          setDeleteDialogOpen(open);
          if (!open) {
            setDeleteConfirmEmail("");
            setDeleteError(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete your account?</DialogTitle>
            <DialogDescription>
              This cannot be undone. Type{" "}
              <span className="font-medium text-foreground">{user?.email}</span>{" "}
              to confirm.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <Label htmlFor="delete-confirm-email">Email</Label>
            <Input
              id="delete-confirm-email"
              type="email"
              autoComplete="off"
              value={deleteConfirmEmail}
              onChange={(e) => setDeleteConfirmEmail(e.target.value)}
              placeholder="your@email.com"
            />
            {deleteError && (
              <p className="text-sm text-destructive">{deleteError}</p>
            )}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={!emailMatchesConfirm || deleteAccountMutation.isPending}
              onClick={() => deleteAccountMutation.mutate()}
            >
              {deleteAccountMutation.isPending ? "Deleting…" : "Delete account"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
