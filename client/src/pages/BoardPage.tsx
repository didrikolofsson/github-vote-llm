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

const STATUS_COLOR: Record<Proposal['status'], string> = {
  open: '#403C34',
  planned: '#00E87A',
  done: '#3A9EFF',
};

// ─── Comment thread ────────────────────────────────────────────────────────────

function CommentThread({ owner, repo, proposalId }: { owner: string; repo: string; proposalId: number }) {
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
    <div style={{ marginTop: 16, borderTop: '1px solid #141414', paddingTop: 16 }}>
      {isLoading ? (
        <p style={{ fontSize: 11, color: '#302E2A', letterSpacing: '0.06em' }}>Loading comments…</p>
      ) : comments.length === 0 ? (
        <p style={{ fontSize: 11, color: '#201E1A', letterSpacing: '0.06em', marginBottom: 12 }}>
          No comments yet. Be the first.
        </p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 16 }}>
          {comments.map((cm: ProposalComment) => (
            <div key={cm.id} style={{ display: 'flex', gap: 12 }}>
              <div
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: '#201E1A',
                  flexShrink: 0,
                  marginTop: 5,
                }}
              />
              <div>
                <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 3 }}>
                  <span style={{ fontSize: 11, color: '#403C34', letterSpacing: '0.05em' }}>
                    {cm.author_name}
                  </span>
                  <span style={{ fontSize: 10, color: '#201E1A', letterSpacing: '0.04em' }}>
                    {relativeTime(cm.created_at)}
                  </span>
                </div>
                <p style={{ fontSize: 12, color: '#8A8476', letterSpacing: '0.02em', lineHeight: 1.6 }}>
                  {cm.body}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Comment form */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Add a comment…"
          rows={3}
          style={{
            width: '100%',
            padding: '8px 10px',
            background: '#0C0C0C',
            border: '1px solid #191919',
            color: '#C4C0AC',
            fontSize: 12,
            letterSpacing: '0.02em',
            outline: 'none',
            resize: 'vertical',
            boxSizing: 'border-box',
            fontFamily: 'inherit',
            lineHeight: 1.5,
          }}
          onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
          onBlur={(e) => (e.target.style.borderColor = '#191919')}
        />
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          <input
            value={author}
            onChange={(e) => setAuthor(e.target.value)}
            placeholder="Your name (optional)"
            style={{
              flex: 1,
              padding: '6px 10px',
              background: '#0C0C0C',
              border: '1px solid #191919',
              color: '#C4C0AC',
              fontSize: 11,
              letterSpacing: '0.03em',
              outline: 'none',
              fontFamily: 'inherit',
            }}
            onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
            onBlur={(e) => (e.target.style.borderColor = '#191919')}
          />
          <button
            onClick={() => body.trim() && post.mutate()}
            disabled={post.isPending || !body.trim()}
            style={{
              padding: '6px 14px',
              background: body.trim() ? '#00E87A' : '#111111',
              color: body.trim() ? '#070707' : '#201E1A',
              fontSize: 10,
              letterSpacing: '0.15em',
              textTransform: 'uppercase',
              fontWeight: 600,
              border: 'none',
              cursor: body.trim() ? 'pointer' : 'not-allowed',
              opacity: post.isPending ? 0.6 : 1,
              transition: 'all 150ms',
              flexShrink: 0,
            }}
          >
            {submitted ? 'Posted' : post.isPending ? '…' : 'Post'}
          </button>
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
    <div
      style={{
        border: '1px solid #141414',
        background: '#0A0A0A',
        transition: 'border-color 200ms',
      }}
      onMouseEnter={(e) => ((e.currentTarget as HTMLElement).style.borderColor = '#1E1E1E')}
      onMouseLeave={(e) => ((e.currentTarget as HTMLElement).style.borderColor = '#141414')}
    >
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '56px 1fr',
          gap: 0,
        }}
      >
        {/* Vote button */}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onVote();
          }}
          disabled={isVoting}
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 2,
            padding: '14px 0',
            background: 'none',
            border: 'none',
            borderRight: '1px solid #141414',
            cursor: 'pointer',
            opacity: isVoting ? 0.5 : 1,
            transition: 'all 150ms',
          }}
          onMouseEnter={(e) => {
            if (!isVoting) {
              const el = e.currentTarget;
              el.style.background = '#0F1A14';
            }
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLElement).style.background = 'none';
          }}
        >
          <svg width="10" height="7" viewBox="0 0 10 7" fill="none">
            <path d="M5 0L10 7H0L5 0Z" fill={isVoting ? '#201E1A' : '#00E87A'} />
          </svg>
          <span
            style={{
              fontSize: 14,
              fontWeight: 600,
              color: '#C4C0AC',
              letterSpacing: '0.02em',
              lineHeight: 1,
            }}
          >
            {proposal.vote_count}
          </span>
        </button>

        {/* Content */}
        <button
          onClick={() => setExpanded((v) => !v)}
          style={{
            display: 'block',
            width: '100%',
            padding: '14px 16px',
            background: 'none',
            border: 'none',
            textAlign: 'left',
            cursor: 'pointer',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 }}>
            <span
              style={{
                fontSize: 13,
                color: '#C4C0AC',
                letterSpacing: '0.02em',
                lineHeight: 1.4,
                fontWeight: 500,
              }}
            >
              {proposal.title}
            </span>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
              <span
                style={{
                  fontSize: 9,
                  letterSpacing: '0.2em',
                  textTransform: 'uppercase',
                  color: STATUS_COLOR[proposal.status],
                  border: `1px solid ${STATUS_COLOR[proposal.status]}40`,
                  padding: '2px 6px',
                }}
              >
                {proposal.status}
              </span>
              <span
                style={{
                  fontSize: 10,
                  color: expanded ? '#403C34' : '#201E1A',
                  letterSpacing: '0.1em',
                  transition: 'color 150ms',
                }}
              >
                {expanded ? '▲' : '▼'}
              </span>
            </div>
          </div>
          {!expanded && proposal.description && (
            <p
              style={{
                marginTop: 5,
                fontSize: 11,
                color: '#302E2A',
                letterSpacing: '0.02em',
                lineHeight: 1.5,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                maxWidth: '100%',
              }}
            >
              {proposal.description}
            </p>
          )}
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 6 }}>
            <span style={{ fontSize: 10, color: '#201E1A', letterSpacing: '0.05em' }}>
              {relativeTime(proposal.created_at)}
            </span>
          </div>
        </button>
      </div>

      {/* Expanded: description + comments */}
      {expanded && (
        <div
          style={{
            padding: '16px 16px 16px 72px',
            borderTop: '1px solid #141414',
          }}
        >
          {proposal.description && (
            <p
              style={{
                fontSize: 12,
                color: '#8A8476',
                letterSpacing: '0.02em',
                lineHeight: 1.7,
                marginBottom: 0,
              }}
            >
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
  onClose,
}: {
  owner: string;
  repo: string;
  onClose: () => void;
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
      onClose();
    },
  });

  const errorMsg =
    create.error instanceof Error ? create.error.message : undefined;
  const canSubmit = title.trim().length >= 3;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.85)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 50,
        padding: 16,
      }}
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="animate-slide-up"
        style={{
          background: '#0C0C0C',
          border: '1px solid #1E1E1E',
          padding: 24,
          width: '100%',
          maxWidth: 480,
        }}
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'baseline',
            marginBottom: 20,
            paddingBottom: 16,
            borderBottom: '1px solid #141414',
          }}
        >
          <span
            style={{
              fontSize: 10,
              letterSpacing: '0.2em',
              textTransform: 'uppercase',
              color: '#302E2A',
            }}
          >
            New Proposal
          </span>
          <button
            onClick={onClose}
            style={{
              fontSize: 18,
              color: '#201E1A',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              lineHeight: 1,
              transition: 'color 150ms',
            }}
            onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6A6458')}
            onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#201E1A')}
          >
            ×
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div>
            <label
              style={{
                display: 'block',
                fontSize: 10,
                letterSpacing: '0.15em',
                textTransform: 'uppercase',
                color: '#302E2A',
                marginBottom: 5,
              }}
            >
              Title *
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Add dark mode toggle"
              maxLength={200}
              style={{
                width: '100%',
                padding: '8px 10px',
                background: '#070707',
                border: '1px solid #191919',
                color: '#C4C0AC',
                fontSize: 12,
                letterSpacing: '0.02em',
                outline: 'none',
                boxSizing: 'border-box',
                fontFamily: 'inherit',
              }}
              onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
              onBlur={(e) => (e.target.style.borderColor = '#191919')}
              autoFocus
            />
          </div>
          <div>
            <label
              style={{
                display: 'block',
                fontSize: 10,
                letterSpacing: '0.15em',
                textTransform: 'uppercase',
                color: '#302E2A',
                marginBottom: 5,
              }}
            >
              Description{' '}
              <span style={{ color: '#201E1A', letterSpacing: '0.08em' }}>(optional)</span>
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe the feature or problem you want solved…"
              rows={4}
              maxLength={5000}
              style={{
                width: '100%',
                padding: '8px 10px',
                background: '#070707',
                border: '1px solid #191919',
                color: '#C4C0AC',
                fontSize: 12,
                letterSpacing: '0.02em',
                outline: 'none',
                resize: 'vertical',
                boxSizing: 'border-box',
                fontFamily: 'inherit',
                lineHeight: 1.5,
              }}
              onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
              onBlur={(e) => (e.target.style.borderColor = '#191919')}
            />
          </div>
        </div>

        {errorMsg && (
          <p style={{ marginTop: 12, fontSize: 11, color: '#FF3A3A', letterSpacing: '0.04em' }}>
            {errorMsg}
          </p>
        )}

        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: 10,
            paddingTop: 16,
            marginTop: 16,
            borderTop: '1px solid #141414',
          }}
        >
          <button
            onClick={onClose}
            style={{
              padding: '8px 14px',
              background: 'none',
              border: 'none',
              fontSize: 10,
              letterSpacing: '0.12em',
              textTransform: 'uppercase',
              color: '#302E2A',
              cursor: 'pointer',
              transition: 'color 150ms',
            }}
            onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6A6458')}
            onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#302E2A')}
          >
            Cancel
          </button>
          <button
            onClick={() => canSubmit && create.mutate()}
            disabled={!canSubmit || create.isPending}
            style={{
              padding: '8px 16px',
              background: canSubmit ? '#00E87A' : '#111111',
              color: canSubmit ? '#070707' : '#201E1A',
              fontSize: 10,
              letterSpacing: '0.15em',
              textTransform: 'uppercase',
              fontWeight: 600,
              border: 'none',
              cursor: canSubmit ? 'pointer' : 'not-allowed',
              opacity: create.isPending ? 0.5 : 1,
              transition: 'all 150ms',
            }}
          >
            {create.isPending ? '…' : 'Submit'}
          </button>
        </div>
      </div>
    </div>
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
      <div style={{ minHeight: '100vh', background: '#070707', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.08em' }}>Invalid board URL.</p>
      </div>
    );
  }

  return (
    <div style={{ minHeight: '100vh', background: '#070707', color: '#C4C0AC' }}>
      {/* Header */}
      <header
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 10,
          background: '#070707',
          borderBottom: '1px solid #141414',
          height: 48,
          display: 'flex',
          alignItems: 'center',
          paddingLeft: 24,
          paddingRight: 24,
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <span
            style={{
              color: '#00E87A',
              fontSize: 11,
              letterSpacing: '0.35em',
              textTransform: 'uppercase',
              fontWeight: 600,
            }}
          >
            vote-llm
          </span>
          <span style={{ fontSize: 10, color: '#191919', letterSpacing: '0.05em' }}>/</span>
          <span style={{ fontSize: 12, color: '#403C34', letterSpacing: '0.05em' }}>
            {owner}/{repo}
          </span>
        </div>
        <button
          onClick={() => setShowNew(true)}
          style={{
            padding: '6px 14px',
            background: '#00E87A',
            color: '#070707',
            fontSize: 10,
            letterSpacing: '0.15em',
            textTransform: 'uppercase',
            fontWeight: 600,
            border: 'none',
            cursor: 'pointer',
            transition: 'opacity 150ms',
          }}
          onMouseEnter={(e) => ((e.target as HTMLElement).style.opacity = '0.85')}
          onMouseLeave={(e) => ((e.target as HTMLElement).style.opacity = '1')}
        >
          + Propose
        </button>
      </header>

      {/* Hero */}
      <div
        style={{
          padding: '48px 24px 32px',
          maxWidth: 720,
          margin: '0 auto',
        }}
      >
        <h1
          style={{
            fontSize: 24,
            fontWeight: 600,
            color: '#C4C0AC',
            letterSpacing: '0.02em',
            marginBottom: 8,
            lineHeight: 1.2,
          }}
        >
          Feature Requests
        </h1>
        <p
          style={{
            fontSize: 12,
            color: '#302E2A',
            letterSpacing: '0.05em',
            lineHeight: 1.6,
          }}
        >
          Vote on the features you want most. The highest-voted proposals shape what gets built next.
        </p>
      </div>

      {/* Content */}
      <div style={{ maxWidth: 720, margin: '0 auto', padding: '0 24px 64px' }}>
        {isLoading ? (
          <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.08em' }}>Loading…</p>
        ) : error ? (
          <div
            style={{
              padding: 20,
              border: '1px solid #2A1414',
              background: '#0D0707',
            }}
          >
            <p style={{ fontSize: 12, color: '#FF3A3A', letterSpacing: '0.04em' }}>
              {error instanceof Error && error.message.includes('403')
                ? 'This board is not public.'
                : `Error: ${error instanceof Error ? error.message : 'unknown'}`}
            </p>
          </div>
        ) : proposals.length === 0 ? (
          <div
            style={{
              padding: '48px 24px',
              border: '1px solid #141414',
              textAlign: 'center',
            }}
          >
            <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.06em', marginBottom: 8 }}>
              No proposals yet.
            </p>
            <p style={{ fontSize: 11, color: '#201E1A', letterSpacing: '0.05em' }}>
              Be the first to suggest a feature.
            </p>
            <button
              onClick={() => setShowNew(true)}
              style={{
                marginTop: 20,
                padding: '8px 20px',
                background: '#00E87A',
                color: '#070707',
                fontSize: 10,
                letterSpacing: '0.15em',
                textTransform: 'uppercase',
                fontWeight: 600,
                border: 'none',
                cursor: 'pointer',
              }}
            >
              + Propose a feature
            </button>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
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

      {showNew && (
        <NewProposalModal owner={owner} repo={repo} onClose={() => setShowNew(false)} />
      )}
    </div>
  );
}
