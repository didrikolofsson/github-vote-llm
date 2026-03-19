import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listBoardProposals,
  createBoardProposal,
  voteProposal,
  listBoardComments,
  createBoardComment,
} from '../client/sdk.gen';
import type { Proposal, ProposalComment } from '../client/types.gen';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const m = Math.floor(diff / 60_000);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (d > 0) return `${d}d ago`;
  if (h > 0) return `${h}h ago`;
  if (m > 0) return `${m}m ago`;
  return 'just now';
}

const STATUS_CLASS: Record<Proposal['status'], string> = {
  open: 'text-muted-foreground border-muted-foreground/25',
  planned: 'text-primary border-primary/25',
  done: 'text-sky-400 border-sky-400/25',
};

// ─── Comment thread ────────────────────────────────────────────────────────────

function CommentThread({
  owner,
  repo,
  proposalId,
}: {
  owner: string;
  repo: string;
  proposalId: number;
}) {
  const qc = useQueryClient();
  const [body, setBody] = useState('');
  const [author, setAuthor] = useState('');
  const [submitted, setSubmitted] = useState(false);

  const { data: comments = [], isLoading } = useQuery({
    queryKey: ['board-comments', owner, repo, proposalId],
    queryFn: () =>
      listBoardComments({ path: { owner, repo, id: proposalId } }).then((r) => r.data ?? []),
  });

  const post = useMutation({
    mutationFn: () =>
      createBoardComment({
        path: { owner, repo, id: proposalId },
        body: { body: body.trim(), author_name: author.trim() || undefined },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['board-comments', owner, repo, proposalId] });
      qc.invalidateQueries({ queryKey: ['board-proposals', owner, repo] });
      setBody('');
      setSubmitted(true);
      setTimeout(() => setSubmitted(false), 2000);
    },
  });

  return (
    <div className="mt-4 pt-4 border-t border-border">
      {isLoading ? (
        <p className="text-[11px] text-muted-foreground tracking-[0.06em]">Loading comments…</p>
      ) : comments.length === 0 ? (
        <p className="text-[11px] text-muted-foreground tracking-[0.06em] mb-3">
          No comments yet. Be the first.
        </p>
      ) : (
        <div className="flex flex-col gap-3 mb-4">
          {comments.map((cm: ProposalComment) => (
            <div key={cm.id} className="flex gap-3">
              <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground flex-shrink-0 mt-1" />
              <div>
                <div className="flex items-baseline gap-2 mb-0.5">
                  <span className="text-[11px] text-muted-foreground tracking-[0.05em]">
                    {cm.author_name}
                  </span>
                  <span className="text-[10px] text-muted-foreground/80 tracking-[0.04em]">
                    {relativeTime(cm.created_at)}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground tracking-[0.02em] leading-relaxed">
                  {cm.body}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="flex flex-col gap-2">
        <Textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Add a comment…"
          rows={3}
          className="text-xs tracking-[0.02em] rounded-none resize-y min-h-[4.5rem]"
        />
        <div className="flex gap-2 items-center">
          <Input
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            placeholder="Your name (optional)"
            className="flex-1 text-[11px] tracking-[0.03em] rounded-none h-9"
          />
          <Button
            onClick={() => body.trim() && post.mutate()}
            disabled={post.isPending || !body.trim()}
            size="sm"
            className="text-[10px] tracking-[0.15em] uppercase flex-shrink-0"
          >
            {submitted ? 'Posted' : post.isPending ? '…' : 'Post'}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── Proposal card ─────────────────────────────────────────────────────────────

function ProposalCard({
  proposal,
  owner,
  repo,
  onVote,
  isVoting,
}: {
  proposal: Proposal;
  owner: string;
  repo: string;
  onVote: () => void;
  isVoting: boolean;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border border-border bg-muted/30 transition-[border-color] duration-200 hover:border-border/80">
      <div className="grid grid-cols-[56px_1fr] gap-0">
        <button
          onClick={(e) => {
            e.stopPropagation();
            onVote();
          }}
          disabled={isVoting}
          className="flex flex-col items-center justify-center gap-0.5 py-3.5 bg-transparent border-none border-r border-border cursor-pointer transition-all duration-150 disabled:opacity-50 hover:bg-primary/10"
        >
          <svg width="10" height="7" viewBox="0 0 10 7" fill="none">
            <path
              d="M5 0L10 7H0L5 0Z"
              className={isVoting ? 'fill-muted-foreground' : 'fill-primary'}
            />
          </svg>
          <span className="text-sm font-semibold text-foreground tracking-[0.02em] leading-none">
            {proposal.vote_count}
          </span>
        </button>

        <button
          onClick={() => setExpanded((v) => !v)}
          className="block w-full py-3.5 px-4 bg-transparent border-none text-left cursor-pointer"
        >
          <div className="flex items-start justify-between gap-3">
            <span className="text-[13px] text-foreground tracking-[0.02em] leading-snug font-medium">
              {proposal.title}
            </span>
            <div className="flex items-center gap-2 flex-shrink-0">
              <span
                className={`text-[9px] tracking-[0.2em] uppercase border py-0.5 px-1.5 ${STATUS_CLASS[proposal.status]}`}
              >
                {proposal.status}
              </span>
              <span
                className={`text-[10px] tracking-[0.1em] transition-colors duration-150 ${
                  expanded ? 'text-muted-foreground' : 'text-muted-foreground/80'
                }`}
              >
                {expanded ? '▲' : '▼'}
              </span>
            </div>
          </div>
          {!expanded && proposal.description && (
            <p className="mt-1 text-[11px] text-muted-foreground tracking-[0.02em] leading-normal overflow-hidden text-ellipsis whitespace-nowrap max-w-full">
              {proposal.description}
            </p>
          )}
          <div className="flex items-center gap-3 mt-1.5">
            <span className="text-[10px] text-muted-foreground tracking-[0.05em]">
              {relativeTime(proposal.created_at)}
            </span>
          </div>
        </button>
      </div>

      {expanded && (
        <div className="pt-4 pb-4 pl-4 pr-4 ml-[72px] border-t border-border">
          {proposal.description && (
            <p className="text-xs text-muted-foreground tracking-[0.02em] leading-relaxed mb-0">
              {proposal.description}
            </p>
          )}
          <CommentThread owner={owner} repo={repo} proposalId={proposal.id} />
        </div>
      )}
    </div>
  );
}

// ─── New proposal modal ────────────────────────────────────────────────────────

function NewProposalModal({
  owner,
  repo,
  open,
  onOpenChange,
}: {
  owner: string;
  repo: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const qc = useQueryClient();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');

  const create = useMutation({
    mutationFn: () =>
      createBoardProposal({
        path: { owner, repo },
        body: { title: title.trim(), description: description.trim() || undefined },
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['board-proposals', owner, repo] });
      onOpenChange(false);
    },
  });

  const errorMsg = create.error instanceof Error ? create.error.message : undefined;
  const canSubmit = title.trim().length >= 3;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-[10px] tracking-[0.2em] uppercase text-muted-foreground font-normal">
            New Proposal
          </DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-3.5">
          <div>
            <Label className="block text-[10px] tracking-[0.15em] uppercase text-muted-foreground mb-1.5">
              Title *
            </Label>
            <Input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Add dark mode toggle"
              maxLength={200}
              className="text-xs tracking-[0.02em] rounded-none h-9"
              autoFocus
            />
          </div>
          <div>
            <Label className="block text-[10px] tracking-[0.15em] uppercase text-muted-foreground mb-1.5">
              Description <span className="text-muted-foreground tracking-[0.08em]">(optional)</span>
            </Label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe the feature or problem you want solved…"
              rows={4}
              maxLength={5000}
              className="text-xs tracking-[0.02em] rounded-none resize-y leading-normal min-h-24"
            />
          </div>
        </div>

        {errorMsg && (
          <p className="text-[11px] text-destructive tracking-[0.04em]">{errorMsg}</p>
        )}

        <DialogFooter className="border-border pt-4">
          <Button variant="ghost" onClick={() => onOpenChange(false)} className="text-[10px] tracking-[0.12em] uppercase">
            Cancel
          </Button>
          <Button
            onClick={() => canSubmit && create.mutate()}
            disabled={!canSubmit || create.isPending}
            className="text-[10px] tracking-[0.15em] uppercase"
          >
            {create.isPending ? '…' : 'Submit'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function BoardPage() {
  const { owner, repo } = useParams<{ owner: string; repo: string }>();
  const qc = useQueryClient();
  const [showNew, setShowNew] = useState(false);

  const {
    data: proposals = [],
    isLoading,
    error,
  } = useQuery({
    queryKey: ['board-proposals', owner, repo],
    queryFn: () =>
      listBoardProposals({ path: { owner: owner!, repo: repo! } }).then((r) => r.data ?? []),
    enabled: !!(owner && repo),
    refetchInterval: 30_000,
  });

  const vote = useMutation({
    mutationFn: (id: number) =>
      voteProposal({ path: { owner: owner!, repo: repo!, id } }),
    onSuccess: (res) => {
      if (!res.data) return;
      qc.setQueryData(
        ['board-proposals', owner, repo],
        (old: Proposal[] = []) =>
          old.map((p) => (p.id === res.data!.id ? res.data! : p)),
      );
    },
  });

  if (!owner || !repo) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <p className="text-xs text-muted-foreground tracking-[0.08em]">Invalid board URL.</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-10 bg-background border-b border-border h-12 flex items-center px-6 justify-between">
        <div className="flex items-center gap-4">
          <span className="text-primary text-[11px] tracking-[0.35em] uppercase font-semibold">
            vote-llm
          </span>
          <span className="text-[10px] text-muted-foreground tracking-[0.05em]">/</span>
          <span className="text-xs text-muted-foreground tracking-[0.05em]">
            {owner}/{repo}
          </span>
        </div>
        <Button
          onClick={() => setShowNew(true)}
          size="sm"
          className="text-[10px] tracking-[0.15em] uppercase"
        >
          + Propose
        </Button>
      </header>

      <div className="py-12 px-6 max-w-[720px] mx-auto">
        <h1 className="text-2xl font-semibold text-foreground tracking-[0.02em] mb-2 leading-tight">
          Feature Requests
        </h1>
        <p className="text-xs text-muted-foreground tracking-[0.05em] leading-relaxed">
          Vote on the features you want most. The highest-voted proposals shape what gets built
          next.
        </p>
      </div>

      <div className="max-w-[720px] mx-auto px-6 pb-16">
        {isLoading ? (
          <p className="text-xs text-muted-foreground tracking-[0.08em]">Loading…</p>
        ) : error ? (
          <div className="p-5 border border-destructive/20 bg-destructive/5">
            <p className="text-xs text-destructive tracking-[0.04em]">
              {error instanceof Error && error.message.includes('403')
                ? 'This board is not public.'
                : `Error: ${error instanceof Error ? error.message : 'unknown'}`}
            </p>
          </div>
        ) : proposals.length === 0 ? (
          <div className="py-12 px-6 border border-border text-center">
            <p className="text-xs text-muted-foreground tracking-[0.06em] mb-2">No proposals yet.</p>
            <p className="text-[11px] text-muted-foreground tracking-[0.05em]">
              Be the first to suggest a feature.
            </p>
            <Button
              onClick={() => setShowNew(true)}
              className="mt-5 text-[10px] tracking-[0.15em] uppercase"
            >
              + Propose a feature
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-0.5">
            {proposals.map((p: Proposal) => (
              <ProposalCard
                key={p.id}
                proposal={p}
                owner={owner}
                repo={repo}
                onVote={() => vote.mutate(p.id)}
                isVoting={vote.isPending && vote.variables === p.id}
              />
            ))}
          </div>
        )}
      </div>

      <NewProposalModal
        owner={owner}
        repo={repo}
        open={showNew}
        onOpenChange={setShowNew}
      />
    </div>
  );
}
