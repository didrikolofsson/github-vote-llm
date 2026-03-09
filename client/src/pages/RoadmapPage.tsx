import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listRepos, listRoadmapItems, updateProposalStatus } from '../client/sdk.gen';
import type { Proposal, RepoConfig } from '../client/types.gen';

// ─── Helpers ──────────────────────────────────────────────────────────────────

const LANE_ORDER: Proposal['status'][] = ['open', 'planned', 'done'];
const LANE_LABEL: Record<Proposal['status'], string> = {
  open: 'Open',
  planned: 'Planned',
  done: 'Done',
};
const LANE_COLOR: Record<Proposal['status'], string> = {
  open: '#403C34',
  planned: '#00E87A',
  done: '#3A9EFF',
};

// ─── Proposal card ────────────────────────────────────────────────────────────

function RoadmapCard({
  proposal,
  owner,
  repo,
}: {
  proposal: Proposal;
  owner: string;
  repo: string;
}) {
  const qc = useQueryClient();
  const [expanded, setExpanded] = useState(false);

  const move = useMutation({
    mutationFn: (status: Proposal['status']) =>
      updateProposalStatus({ path: { owner, repo, id: proposal.id }, body: { status } }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['roadmap', owner, repo] });
    },
  });

  const nextStatus: Partial<Record<Proposal['status'], Proposal['status']>> = {
    open: 'planned',
    planned: 'done',
  };
  const prevStatus: Partial<Record<Proposal['status'], Proposal['status']>> = {
    planned: 'open',
    done: 'planned',
  };

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
      <button
        onClick={() => setExpanded((v) => !v)}
        style={{
          display: 'block',
          width: '100%',
          padding: '12px 14px',
          background: 'none',
          border: 'none',
          textAlign: 'left',
          cursor: 'pointer',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
          {/* Vote count */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 3,
              flexShrink: 0,
              paddingTop: 2,
            }}
          >
            <svg width="7" height="5" viewBox="0 0 7 5" fill="none">
              <path d="M3.5 0L7 5H0L3.5 0Z" fill="#201E1A" />
            </svg>
            <span style={{ fontSize: 12, fontWeight: 600, color: '#403C34', letterSpacing: '0.02em' }}>
              {proposal.vote_count}
            </span>
          </div>
          <span
            style={{
              fontSize: 12,
              color: '#C4C0AC',
              letterSpacing: '0.02em',
              lineHeight: 1.4,
              fontWeight: 500,
              flex: 1,
            }}
          >
            {proposal.title}
          </span>
          <span style={{ fontSize: 10, color: '#201E1A', letterSpacing: '0.1em', flexShrink: 0 }}>
            {expanded ? '▲' : '▼'}
          </span>
        </div>
      </button>

      {expanded && (
        <div style={{ padding: '0 14px 14px' }}>
          {proposal.description && (
            <p
              style={{
                fontSize: 11,
                color: '#403C34',
                letterSpacing: '0.02em',
                lineHeight: 1.6,
                marginBottom: 12,
                borderTop: '1px solid #111111',
                paddingTop: 12,
              }}
            >
              {proposal.description}
            </p>
          )}
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {prevStatus[proposal.status] && (
              <button
                onClick={() => move.mutate(prevStatus[proposal.status]!)}
                disabled={move.isPending}
                style={{
                  padding: '4px 10px',
                  background: 'none',
                  border: '1px solid #191919',
                  fontSize: 9,
                  letterSpacing: '0.15em',
                  textTransform: 'uppercase',
                  color: '#302E2A',
                  cursor: 'pointer',
                  opacity: move.isPending ? 0.4 : 1,
                  transition: 'all 150ms',
                }}
                onMouseEnter={(e) => {
                  const el = e.currentTarget;
                  el.style.borderColor = '#302E2A';
                  el.style.color = '#6A6458';
                }}
                onMouseLeave={(e) => {
                  const el = e.currentTarget;
                  el.style.borderColor = '#191919';
                  el.style.color = '#302E2A';
                }}
              >
                ← {LANE_LABEL[prevStatus[proposal.status]!]}
              </button>
            )}
            {nextStatus[proposal.status] && (
              <button
                onClick={() => move.mutate(nextStatus[proposal.status]!)}
                disabled={move.isPending}
                style={{
                  padding: '4px 10px',
                  background: 'none',
                  border: `1px solid ${LANE_COLOR[nextStatus[proposal.status]!]}40`,
                  fontSize: 9,
                  letterSpacing: '0.15em',
                  textTransform: 'uppercase',
                  color: LANE_COLOR[nextStatus[proposal.status]!],
                  cursor: 'pointer',
                  opacity: move.isPending ? 0.4 : 1,
                  transition: 'all 150ms',
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLElement).style.opacity = '0.7';
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLElement).style.opacity = move.isPending ? '0.4' : '1';
                }}
              >
                {LANE_LABEL[nextStatus[proposal.status]!]} →
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Kanban lane ──────────────────────────────────────────────────────────────

function KanbanLane({
  status,
  proposals,
  owner,
  repo,
}: {
  status: Proposal['status'];
  proposals: Proposal[];
  owner: string;
  repo: string;
}) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 0, minWidth: 0 }}>
      {/* Lane header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          marginBottom: 10,
          paddingBottom: 10,
          borderBottom: `1px solid ${LANE_COLOR[status]}30`,
        }}
      >
        <span
          style={{
            fontSize: 9,
            letterSpacing: '0.25em',
            textTransform: 'uppercase',
            color: LANE_COLOR[status],
            fontWeight: 600,
          }}
        >
          {LANE_LABEL[status]}
        </span>
        <span
          style={{
            fontSize: 10,
            color: '#201E1A',
            letterSpacing: '0.05em',
          }}
        >
          {proposals.length}
        </span>
      </div>

      {/* Cards */}
      {proposals.length === 0 ? (
        <div
          style={{
            padding: '16px',
            border: '1px dashed #141414',
            textAlign: 'center',
          }}
        >
          <span style={{ fontSize: 10, color: '#191919', letterSpacing: '0.08em' }}>empty</span>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {proposals.map((p) => (
            <RoadmapCard key={p.id} proposal={p} owner={owner} repo={repo} />
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function RoadmapPage() {
  const qc = useQueryClient();
  const [selectedIdx, setSelectedIdx] = useState(0);

  const { data: repos = [], isLoading: reposLoading } = useQuery({
    queryKey: ['repos'],
    queryFn: () => listRepos().then((r) => r.data ?? []),
  });

  const selectedRepo: RepoConfig | undefined = repos[selectedIdx];
  const owner = selectedRepo?.owner;
  const repo = selectedRepo?.repo;

  const { data: proposals = [], isLoading: proposalsLoading } = useQuery({
    queryKey: ['roadmap', owner, repo],
    queryFn: () =>
      listRoadmapItems({ path: { owner: owner!, repo: repo! } }).then((r) => r.data ?? []),
    enabled: !!(owner && repo),
    refetchInterval: 30_000,
  });

  const byStatus = (status: Proposal['status']) =>
    proposals.filter((p) => p.status === status).sort((a, b) => b.vote_count - a.vote_count);

  if (reposLoading) {
    return <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.08em' }}>Loading…</p>;
  }

  return (
    <div className="animate-slide-up">
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'baseline',
          justifyContent: 'space-between',
          marginBottom: 24,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 12 }}>
          <span
            style={{
              fontSize: 10,
              letterSpacing: '0.25em',
              textTransform: 'uppercase',
              color: '#00E87A',
            }}
          >
            Roadmap
          </span>
          {proposals.length > 0 && (
            <span style={{ fontSize: 10, color: '#302E2A', letterSpacing: '0.05em' }}>
              {proposals.length} proposals
            </span>
          )}
        </div>
        <button
          onClick={() => qc.invalidateQueries({ queryKey: ['roadmap', owner, repo] })}
          style={{
            fontSize: 10,
            letterSpacing: '0.12em',
            textTransform: 'uppercase',
            color: '#302E2A',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            transition: 'color 150ms',
          }}
          onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6A6458')}
          onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#302E2A')}
        >
          Refresh
        </button>
      </div>

      {/* Repo selector */}
      {repos.length === 0 ? (
        <div
          style={{
            padding: '32px 0',
            borderTop: '1px solid #191919',
            borderBottom: '1px solid #191919',
          }}
        >
          <p
            style={{
              fontSize: 12,
              color: '#302E2A',
              letterSpacing: '0.06em',
              textAlign: 'center',
            }}
          >
            no repos configured
          </p>
          <p
            style={{
              marginTop: 8,
              fontSize: 11,
              color: '#201E1A',
              letterSpacing: '0.06em',
              textAlign: 'center',
            }}
          >
            add a repo in Config to start tracking proposals
          </p>
        </div>
      ) : (
        <>
          {repos.length > 1 && (
            <div
              style={{
                display: 'flex',
                gap: 0,
                marginBottom: 24,
                borderBottom: '1px solid #141414',
              }}
            >
              {repos.map((r: RepoConfig, i: number) => (
                <button
                  key={r.id}
                  onClick={() => setSelectedIdx(i)}
                  style={{
                    padding: '8px 16px',
                    background: 'none',
                    border: 'none',
                    borderBottom: i === selectedIdx ? '1px solid #00E87A' : '1px solid transparent',
                    fontSize: 11,
                    letterSpacing: '0.05em',
                    color: i === selectedIdx ? '#C4C0AC' : '#302E2A',
                    cursor: 'pointer',
                    marginBottom: -1,
                    transition: 'color 150ms',
                  }}
                >
                  {r.owner}/{r.repo}
                </button>
              ))}
            </div>
          )}

          {proposalsLoading ? (
            <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.08em' }}>Loading…</p>
          ) : (
            <div>
              {/* Board link */}
              {selectedRepo?.is_board_public && (
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    marginBottom: 20,
                    padding: '10px 14px',
                    background: '#080E0B',
                    border: '1px solid #001A0D',
                  }}
                >
                  <span
                    style={{
                      width: 4,
                      height: 4,
                      borderRadius: '50%',
                      background: '#00E87A',
                      display: 'inline-block',
                      flexShrink: 0,
                    }}
                  />
                  <span style={{ fontSize: 11, color: '#302E2A', letterSpacing: '0.04em', flex: 1 }}>
                    Community board is public
                  </span>
                  <button
                    onClick={() => {
                      const url = `${window.location.origin}/board/${owner}/${repo}`;
                      navigator.clipboard.writeText(url);
                    }}
                    style={{
                      fontSize: 10,
                      letterSpacing: '0.12em',
                      textTransform: 'uppercase',
                      color: '#00E87A',
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      opacity: 0.7,
                      transition: 'opacity 150ms',
                    }}
                    onMouseEnter={(e) => ((e.target as HTMLElement).style.opacity = '1')}
                    onMouseLeave={(e) => ((e.target as HTMLElement).style.opacity = '0.7')}
                  >
                    Copy link
                  </button>
                  <a
                    href={`/board/${owner}/${repo}`}
                    target="_blank"
                    rel="noreferrer"
                    style={{
                      fontSize: 10,
                      letterSpacing: '0.12em',
                      textTransform: 'uppercase',
                      color: '#00E87A',
                      textDecoration: 'none',
                      opacity: 0.7,
                      transition: 'opacity 150ms',
                    }}
                    onMouseEnter={(e) => ((e.target as HTMLElement).style.opacity = '1')}
                    onMouseLeave={(e) => ((e.target as HTMLElement).style.opacity = '0.7')}
                  >
                    View →
                  </a>
                </div>
              )}

              {/* Kanban */}
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(3, 1fr)',
                  gap: 16,
                  alignItems: 'start',
                }}
              >
                {LANE_ORDER.map((status) => (
                  <KanbanLane
                    key={status}
                    status={status}
                    proposals={byStatus(status)}
                    owner={owner!}
                    repo={repo!}
                  />
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
