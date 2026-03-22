import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listMyOrganizations,
  getGitHubStatus,
  getGitHubAuthorizeUrl,
  listOrgRepositories,
  listAvailableRepositories,
  addRepository,
  removeRepository,
  listOrgMembers,
  inviteMember,
  removeMember,
  updateMemberRole,
  type OrgRepository,
} from '@/lib/api';
import { LayoutGrid, Github, Plus, Trash2, Users, Mail } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';

export default function OrganizationDashboardPage() {
  const queryClient = useQueryClient();
  const [addRepoOpen, setAddRepoOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const connected = params.get('github_connected');
    const error = params.get('github_error');
    if (connected === '1' || error) {
      queryClient.invalidateQueries({ queryKey: ['github-status'] });
      const url = new URL(window.location.href);
      url.searchParams.delete('github_connected');
      url.searchParams.delete('github_error');
      window.history.replaceState({}, '', url.pathname);
    }
  }, [queryClient]);

  const { data: orgs = [], isLoading: orgsLoading } = useQuery({
    queryKey: ['organizations'],
    queryFn: () => listMyOrganizations(),
  });

  const org = orgs[0];
  const orgId = org?.id;

  const { data: ghStatus, isLoading: ghLoading } = useQuery({
    queryKey: ['github-status'],
    queryFn: () => getGitHubStatus(),
    enabled: !!orgId,
  });

  const { data: repos = [], isLoading: reposLoading } = useQuery({
    queryKey: ['organizations', orgId, 'repositories'],
    queryFn: () => listOrgRepositories(orgId!),
    enabled: !!orgId,
  });

  const { data: members = [], isLoading: membersLoading } = useQuery({
    queryKey: ['organizations', orgId, 'members'],
    queryFn: () => listOrgMembers(orgId!),
    enabled: !!orgId,
  });

  const connectGitHub = useMutation({
    mutationFn: async () => {
      const { authorize_url } = await getGitHubAuthorizeUrl();
      window.location.href = authorize_url;
    },
  });

  const addRepo = useMutation({
    mutationFn: ({ owner, repo }: { owner: string; repo: string }) =>
      addRepository(orgId!, owner, repo),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations', orgId, 'repositories'] });
      setAddRepoOpen(false);
    },
  });

  const removeRepo = useMutation({
    mutationFn: ({ owner, repo }: { owner: string; repo: string }) =>
      removeRepository(orgId!, owner, repo),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations', orgId, 'repositories'] });
    },
  });

  const inviteMemberMutation = useMutation({
    mutationFn: (email: string) => inviteMember(orgId!, email),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations', orgId, 'members'] });
      setInviteEmail('');
    },
  });

  const removeMemberMutation = useMutation({
    mutationFn: (userId: number) => removeMember(orgId!, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations', orgId, 'members'] });
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: number; role: 'owner' | 'member' }) =>
      updateMemberRole(orgId!, userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations', orgId, 'members'] });
    },
  });

  if (orgsLoading) {
    return (
      <div className="animate-slide-up flex flex-col gap-4">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-[180px] w-full" />
        <Skeleton className="h-[180px] w-full" />
      </div>
    );
  }

  return (
    <div className="animate-slide-up">
      <div className="mb-8">
        <h1 className="text-[28px] font-bold text-foreground">
          {org?.name ?? 'Dashboard'}
        </h1>
        <p className="text-[15px] text-muted-foreground mt-1">
          Manage your organization and repositories.
        </p>
      </div>

      {/* GitHub connection */}
      <Card className="mb-8">
        <CardHeader className="pb-3">
          <CardTitle className="text-[15px] flex items-center gap-2">
            <Github className="size-4" />
            GitHub connection
          </CardTitle>
        </CardHeader>
        <CardContent>
          {ghLoading ? (
            <Skeleton className="h-4 w-48" />
          ) : ghStatus?.connected ? (
            <p className="text-sm text-muted-foreground">
              Connected as <span className="font-medium text-foreground">@{ghStatus.login}</span>
            </p>
          ) : (
            <div className="flex flex-col gap-3">
              <p className="text-sm text-muted-foreground">
                Connect your GitHub account to add repositories.
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

      {/* Repositories */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-[15px] flex items-center gap-2">
              <LayoutGrid className="size-4" />
              Repositories
            </CardTitle>
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
            <div className="py-8 text-center rounded-lg bg-muted/50">
              <p className="text-sm text-muted-foreground">
                No repositories yet. Connect GitHub and add your first repo.
              </p>
            </div>
          ) : (
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
                    onClick={() => removeRepo.mutate({ owner: r.owner, repo: r.repo })}
                    disabled={removeRepo.isPending}
                  >
                    <Trash2 />
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      {/* Members */}
      <Card className="mt-8">
        <CardHeader className="pb-3">
          <CardTitle className="text-[15px] flex items-center gap-2">
            <Users className="size-4" />
            Members
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2 mb-4">
            <Input
              type="email"
              placeholder="Email to invite"
              value={inviteEmail}
              onChange={(e) => setInviteEmail(e.target.value)}
            />
            <Button
              size="sm"
              onClick={() => inviteEmail && inviteMemberMutation.mutate(inviteEmail)}
              disabled={!inviteEmail || inviteMemberMutation.isPending}
            >
              <Mail data-icon="inline-start" />
              Invite
            </Button>
          </div>
          {membersLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : (
            <ul className="flex flex-col gap-2">
              {members.map((m) => (
                <li
                  key={m.user_id}
                  className="flex items-center justify-between py-2 px-3 rounded-lg bg-muted/30"
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-foreground">{m.email}</span>
                    <Badge variant="secondary">{m.role}</Badge>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() =>
                        updateRoleMutation.mutate({
                          userId: m.user_id,
                          role: m.role === 'member' ? 'owner' : 'member',
                        })
                      }
                      disabled={updateRoleMutation.isPending}
                    >
                      {m.role === 'member' ? 'Make owner' : 'Make member'}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-8 text-muted-foreground hover:text-destructive"
                      onClick={() => removeMemberMutation.mutate(m.user_id)}
                      disabled={removeMemberMutation.isPending}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

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

  const { data, isLoading } = useQuery({
    queryKey: ['available-repos', orgId, page],
    queryFn: () => listAvailableRepositories(orgId!, page),
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
            Select a repository from your GitHub account to add to this organization.
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-[320px] overflow-y-auto mt-4">
          {isLoading ? (
            <div className="flex flex-col gap-2 py-4">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : repos.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">
              No repositories found.
            </p>
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
                      onClick={() => !alreadyAdded && onAdd({ owner: r.owner, repo: r.repo })}
                      disabled={alreadyAdded || adding}
                    >
                      <span className="font-mono">{key}</span>
                      {alreadyAdded ? (
                        <span className="text-xs text-muted-foreground">Added</span>
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
              <Button variant="outline" size="sm" onClick={() => setPage((p) => p + 1)}>
                Load more
              </Button>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
