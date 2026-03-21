import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listRepos,
  listRoadmapItems,
  updateProposalStatus,
  type Proposal,
  type RepoConfig,
} from '@/lib/api';

// ─── Helpers ──────────────────────────────────────────────────────────────────

const LANE_ORDER: Proposal['status'][] = ['open', 'planned', 'done'];
const LANE_LABEL: Record<Proposal['status'], string> = {
  open: 'Open',
  planned: 'Planned',
  done: 'Done',
};
const LANE_CLASS: Record<Proposal['status'], string> = {
  open: 'text-gray-400 border-gray-400/25',
  planned: 'text-emerald-400 border-emerald-400/25',
  done: 'text-sky-400 border-sky-400/25',
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
      updateProposalStatus(owner, repo, proposal.id, { status }),
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
    <div className="border border-gray-800 bg-gray-900 transition-[border-color] duration-200 hover:border-gray-700">
      <button
        onClick={() => setExpanded((v) => !v)}
        className="block w-full py-3 px-3.5 bg-transparent border-none text-left cursor-pointer"
      >
        <div className="flex items-start gap-2.5">
          <div className="flex items-center gap-0.5 flex-shrink-0 pt-0.5">
            <svg width="7" height="5" viewBox="0 0 7 5" fill="none">
              <path d="M3.5 0L7 5H0L3.5 0Z" className="fill-gray-500" />
            </svg>
            <span className="text-xs font-semibold text-gray-400 tracking-[0.02em]">
              {proposal.vote_count}
            </span>
          </div>
          <span className="text-xs text-gray-100 tracking-[0.02em] leading-snug font-medium flex-1">
            {proposal.title}
          </span>
          <span className="text-[10px] text-gray-500 tracking-[0.1em] flex-shrink-0">
            {expanded ? '▲' : '▼'}
          </span>
        </div>
      </button>

      {expanded && (
        <div className="px-3.5 pb-3.5">
          {proposal.description && (
            <p className="text-[11px] text-gray-400 tracking-[0.02em] leading-relaxed mb-3 border-t border-gray-900 pt-3">
              {proposal.description}
            </p>
          )}
          <div className="flex gap-2 flex-wrap">
            {prevStatus[proposal.status] && (
              <button
                onClick={() => move.mutate(prevStatus[proposal.status]!)}
                disabled={move.isPending}
                className="py-1 px-2.5 bg-transparent border border-gray-800 text-gray-500 hover:border-gray-500 hover:text-gray-300 text-[9px] tracking-[0.15em] uppercase cursor-pointer transition-all duration-150 disabled:opacity-40"
              >
                ← {LANE_LABEL[prevStatus[proposal.status]!]}
              </button>
            )}
            {nextStatus[proposal.status] && (
              <button
                onClick={() => move.mutate(nextStatus[proposal.status]!)}
                disabled={move.isPending}
                className={`py-1 px-2.5 bg-transparent border text-[9px] tracking-[0.15em] uppercase cursor-pointer transition-opacity duration-150 disabled:opacity-40 hover:opacity-70 ${LANE_CLASS[nextStatus[proposal.status]!]}`}
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
  const borderColor =
    status === 'open'
      ? 'border-gray-400/20'
      : status === 'planned'
        ? 'border-emerald-400/20'
        : 'border-sky-400/20';

  return (
    <div className="flex flex-col gap-0 min-w-0">
      <div className={`flex items-center gap-2 mb-2.5 pb-2.5 border-b ${borderColor}`}>
        <span
          className={`text-[9px] tracking-[0.25em] uppercase font-semibold ${LANE_CLASS[status].split(' ')[0]}`}
        >
          {LANE_LABEL[status]}
        </span>
        <span className="text-[10px] text-gray-500 tracking-[0.05em]">{proposals.length}</span>
      </div>

      {proposals.length === 0 ? (
        <div className="p-4 border border-dashed border-gray-800 text-center">
          <span className="text-[10px] text-gray-600 tracking-[0.08em]">empty</span>
        </div>
      ) : (
        <div className="flex flex-col gap-0.5">
          {proposals.map((p) => (
            <RoadmapCard key={p.id} proposal={p} owner={owner} repo={repo} />
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Main page ──────────────────────────────────────────────────────────────────

export default function RoadmapPage() {
  const qc = useQueryClient();
  const [selectedIdx, setSelectedIdx] = useState(0);

  const { data: repos = [], isLoading: reposLoading } = useQuery({
    queryKey: ['repos'],
    queryFn: () => listRepos(),
  });

  const selectedRepo: RepoConfig | undefined = repos[selectedIdx];
  const owner = selectedRepo?.owner;
  const repo = selectedRepo?.repo;

  const { data: proposals = [], isLoading: proposalsLoading } = useQuery({
    queryKey: ['roadmap', owner, repo],
    queryFn: () => listRoadmapItems(owner!, repo!),
    enabled: !!(owner && repo),
    refetchInterval: 30_000,
  });

  const byStatus = (status: Proposal['status']) =>
    proposals.filter((p) => p.status === status).sort((a, b) => b.vote_count - a.vote_count);

  if (reposLoading) {
    return <p className="text-xs text-gray-500 tracking-[0.08em]">Loading…</p>;
  }

  return (
    <div className="animate-slide-up">
      {/* Header */}
      <div className="flex items-baseline justify-between mb-6">
        <div className="flex items-baseline gap-3">
          <span className="text-[10px] tracking-[0.25em] uppercase text-emerald-400">Roadmap</span>
          {proposals.length > 0 && (
            <span className="text-[10px] text-gray-500 tracking-[0.05em]">
              {proposals.length} proposals
            </span>
          )}
        </div>
        <button
          onClick={() => qc.invalidateQueries({ queryKey: ['roadmap', owner, repo] })}
          className="text-[10px] tracking-[0.12em] uppercase text-gray-500 hover:text-gray-300 bg-transparent border-none cursor-pointer transition-colors duration-150"
        >
          Refresh
        </button>
      </div>

      {/* Repo selector */}
      {repos.length === 0 ? (
        <div className="py-8 border-t border-b border-gray-800">
          <p className="text-xs text-gray-500 tracking-[0.06em] text-center">
            no repos configured
          </p>
          <p className="mt-2 text-[11px] text-gray-500 tracking-[0.06em] text-center">
            add a repo in Config to start tracking proposals
          </p>
        </div>
      ) : (
        <>
          {repos.length > 1 && (
            <div className="flex gap-0 mb-6 border-b border-gray-800">
              {repos.map((r: RepoConfig, i: number) => (
                <button
                  key={r.id}
                  onClick={() => setSelectedIdx(i)}
                  className={`py-2 px-4 bg-transparent border-none border-b-2 -mb-px text-[11px] tracking-[0.05em] cursor-pointer transition-colors duration-150 ${
                    i === selectedIdx
                      ? 'border-emerald-400 text-gray-100'
                      : 'border-transparent text-gray-500'
                  }`}
                >
                  {r.owner}/{r.repo}
                </button>
              ))}
            </div>
          )}

          {proposalsLoading ? (
            <p className="text-xs text-gray-500 tracking-[0.08em]">Loading…</p>
          ) : (
            <div>
              {/* Board link */}
              {selectedRepo?.is_board_public && (
                <div className="flex items-center gap-2.5 mb-5 py-2.5 px-3.5 bg-emerald-400/5 border border-emerald-400/10">
                  <span className="w-1 h-1 rounded-full bg-emerald-400 flex-shrink-0 inline-block" />
                  <span className="text-[11px] text-gray-500 tracking-[0.04em] flex-1">
                    Community board is public
                  </span>
                  <button
                    onClick={() => {
                      const url = `${window.location.origin}/board/${owner}/${repo}`;
                      navigator.clipboard.writeText(url);
                    }}
                    className="text-[10px] tracking-[0.12em] uppercase text-emerald-400 bg-transparent border-none cursor-pointer opacity-70 transition-opacity duration-150 hover:opacity-100"
                  >
                    Copy link
                  </button>
                  <a
                    href={`/board/${owner}/${repo}`}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[10px] tracking-[0.12em] uppercase text-emerald-400 no-underline opacity-70 transition-opacity duration-150 hover:opacity-100"
                  >
                    View →
                  </a>
                </div>
              )}

              {/* Kanban */}
              <div className="grid grid-cols-3 gap-4 items-start">
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
