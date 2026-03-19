import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getRun, cancelRun, retryRun } from '../client/sdk.gen';
import StatusBadge from '../components/StatusBadge';

function fmt(iso: string) {
  return new Date(iso).toLocaleString('en-US', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <span className="text-[10px] tracking-[0.2em] uppercase text-gray-500 mb-1 block">
        {label}
      </span>
      <div className="text-xs text-gray-100 tracking-[0.03em]">{children}</div>
    </div>
  );
}

export default function RunDetailPage() {
  const { id } = useParams<{ id: string }>();
  const runId = Number(id);
  const qc = useQueryClient();

  const { data: run, isLoading, error } = useQuery({
    queryKey: ['runs', runId],
    queryFn: () => getRun({ path: { id: runId } }).then((r) => r.data),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === 'pending' || status === 'in_progress' ? 5_000 : false;
    },
  });

  const cancel = useMutation({
    mutationFn: () => cancelRun({ path: { id: runId } }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs', runId] }),
  });

  const retry = useMutation({
    mutationFn: () => retryRun({ path: { id: runId } }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs', runId] }),
  });

  if (isLoading) {
    return <p className="text-xs text-gray-500 tracking-[0.08em]">Loading…</p>;
  }

  if (error || !run) {
    return (
      <div>
        <Link
          to="/runs"
          className="text-[10px] tracking-[0.12em] uppercase text-gray-500 no-underline transition-colors duration-150 hover:text-gray-300"
        >
          ← Runs
        </Link>
        <p className="mt-6 text-xs text-red-500 tracking-[0.04em]">run not found.</p>
      </div>
    );
  }

  return (
    <div className="animate-slide-up max-w-[640px]">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2.5 mb-8">
        <Link
          to="/runs"
          className="text-[10px] tracking-[0.12em] uppercase text-gray-500 no-underline transition-colors duration-150 hover:text-gray-300"
        >
          Runs
        </Link>
        <span className="text-gray-600 text-[10px]">/</span>
        <span className="text-[10px] tracking-[0.1em] text-gray-400">#{run.id}</span>
        <StatusBadge status={run.status} />
      </div>

      {/* Fields */}
      <div className="grid grid-cols-2 gap-5 gap-x-8 py-6 border-t border-b border-gray-800 mb-6">
        <Field label="Repository">
          {run.owner}/{run.repo}
        </Field>
        <Field label="Issue">#{run.issue_number}</Field>
        <Field label="Branch">
          {run.branch ? (
            <span className="text-[11px] text-gray-300 tracking-[0.02em]">{run.branch}</span>
          ) : (
            <span className="text-gray-500">—</span>
          )}
        </Field>
        <Field label="Pull Request">
          {run.pr_url ? (
            <a
              href={run.pr_url}
              target="_blank"
              rel="noreferrer"
              className="text-sky-400 no-underline text-xs tracking-[0.03em] transition-colors duration-150 hover:text-sky-400/80"
            >
              → View PR
            </a>
          ) : (
            <span className="text-gray-500">—</span>
          )}
        </Field>
        <Field label="Created">{fmt(run.created_at)}</Field>
        <Field label="Updated">{fmt(run.updated_at)}</Field>
      </div>

      {/* Error block */}
      {run.error && (
        <div className="mb-6 p-4 bg-red-500/5 border-l-2 border-red-500">
          <span className="block text-[10px] tracking-[0.2em] uppercase text-red-500 mb-2">Error</span>
          <pre className="text-[11px] text-red-500/90 whitespace-pre-wrap break-words leading-relaxed m-0">
            {run.error}
          </pre>
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-3">
        {(run.status === 'failed' || run.status === 'cancelled') && (
          <button
            onClick={() => retry.mutate()}
            disabled={retry.isPending}
            className="py-2 px-4 bg-sky-400 text-gray-950 text-[10px] tracking-[0.15em] uppercase font-semibold border-none cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 transition-opacity duration-150 hover:opacity-85"
          >
            {retry.isPending ? 'Retrying…' : 'Retry'}
          </button>
        )}
        {(run.status === 'pending' || run.status === 'in_progress') && (
          <button
            onClick={() => cancel.mutate()}
            disabled={cancel.isPending}
            className="py-2 px-4 bg-gray-900 text-gray-400 hover:text-gray-100 text-[10px] tracking-[0.15em] uppercase font-semibold border border-gray-800 cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 transition-colors duration-150"
          >
            {cancel.isPending ? 'Cancelling…' : 'Cancel'}
          </button>
        )}
      </div>
    </div>
  );
}
