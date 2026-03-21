import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { listRuns, cancelRun, retryRun, type Run } from '@/lib/api';
import StatusBadge from '../components/StatusBadge';

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

const TH_CLASS =
  'pb-2 pr-6 text-[10px] tracking-[0.15em] uppercase text-gray-500 font-normal text-left border-b border-gray-800 whitespace-nowrap';

export default function RunsPage() {
  const qc = useQueryClient();

  const { data: runs, isLoading, error } = useQuery({
    queryKey: ['runs'],
    queryFn: () => listRuns(),
    refetchInterval: 15_000,
  });

  const cancel = useMutation({
    mutationFn: (id: number) => cancelRun(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs'] }),
  });

  const retry = useMutation({
    mutationFn: (id: number) => retryRun(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs'] }),
  });

  if (isLoading) {
    return <p className="text-xs text-gray-500 tracking-[0.08em]">Loading…</p>;
  }

  if (error) {
    return (
      <p className="text-xs text-red-500 tracking-[0.04em]">
        error: {error instanceof Error ? error.message : 'unknown'}
      </p>
    );
  }

  return (
    <div className="animate-slide-up">
      <div className="flex items-baseline justify-between mb-6">
        <div className="flex items-baseline gap-3">
          <span className="text-[10px] tracking-[0.25em] uppercase text-emerald-400">Runs</span>
          {runs && (
            <span className="text-[10px] text-gray-500 tracking-[0.05em]">{runs.length} total</span>
          )}
        </div>
        <button
          onClick={() => qc.invalidateQueries({ queryKey: ['runs'] })}
          className="text-[10px] tracking-[0.12em] uppercase text-gray-500 hover:text-gray-300 bg-transparent border-none cursor-pointer transition-colors duration-150"
        >
          Refresh
        </button>
      </div>

      {runs?.length === 0 ? (
        <p className="text-xs text-gray-500 tracking-[0.06em]">no runs yet.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-xs">
            <thead>
              <tr>
                <th className={TH_CLASS}>ID</th>
                <th className={TH_CLASS}>Repository</th>
                <th className={TH_CLASS}>Issue</th>
                <th className={TH_CLASS}>Status</th>
                <th className={TH_CLASS}>Branch</th>
                <th className={TH_CLASS}>Updated</th>
                <th className={`${TH_CLASS} pr-0`}></th>
              </tr>
            </thead>
            <tbody>
              {runs?.map((run: Run, i: number) => (
                <tr
                  key={run.id}
                  className="border-b border-gray-900 hover:bg-gray-900 transition-colors"
                  style={{ animationDelay: `${i * 30}ms` }}
                >
                  <td className="py-2.5 pr-6">
                    <Link
                      to={`/runs/${run.id}`}
                      className="text-gray-400 no-underline tracking-[0.04em] transition-colors duration-150 hover:text-gray-100"
                    >
                      #{run.id}
                    </Link>
                  </td>
                  <td className="py-2.5 pr-6 text-gray-100 tracking-[0.02em]">
                    {run.owner}/{run.repo}
                  </td>
                  <td className="py-2.5 pr-6 text-gray-400 tracking-[0.04em]">
                    #{run.issue_number}
                  </td>
                  <td className="py-2.5 pr-6">
                    <StatusBadge status={run.status} />
                  </td>
                  <td
                    className="py-2.5 pr-6 text-gray-500 text-[11px] tracking-[0.02em] max-w-[220px] overflow-hidden text-ellipsis whitespace-nowrap"
                  >
                    {run.branch ?? '—'}
                  </td>
                  <td className="py-2.5 pr-6 text-gray-500 text-[11px] tracking-[0.04em] whitespace-nowrap">
                    {relativeTime(run.updated_at)}
                  </td>
                  <td className="py-2.5 text-right">
                    {(run.status === 'failed' || run.status === 'cancelled') && (
                      <button
                        onClick={() => retry.mutate(run.id)}
                        disabled={retry.isPending}
                        className="text-[10px] tracking-[0.12em] uppercase text-sky-400 bg-transparent border-none cursor-pointer transition-colors duration-150 disabled:opacity-40 hover:text-sky-400-400/90"
                      >
                        Retry
                      </button>
                    )}
                    {(run.status === 'pending' || run.status === 'in_progress') && (
                      <button
                        onClick={() => cancel.mutate(run.id)}
                        disabled={cancel.isPending}
                        className="text-[10px] tracking-[0.12em] uppercase text-gray-500 hover:text-gray-300 bg-transparent border-none cursor-pointer transition-colors duration-150 disabled:opacity-40"
                      >
                        Cancel
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
