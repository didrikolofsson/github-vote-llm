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

const LABEL_STYLE: React.CSSProperties = {
  fontSize: 10,
  letterSpacing: '0.2em',
  textTransform: 'uppercase',
  color: '#302E2A',
  marginBottom: 4,
  display: 'block',
};

const VALUE_STYLE: React.CSSProperties = {
  fontSize: 12,
  color: '#C4C0AC',
  letterSpacing: '0.03em',
};

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <span style={LABEL_STYLE}>{label}</span>
      <div style={VALUE_STYLE}>{children}</div>
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
    return (
      <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.08em' }}>
        Loading…
      </p>
    );
  }

  if (error || !run) {
    return (
      <div>
        <Link
          to="/runs"
          style={{
            fontSize: 10,
            letterSpacing: '0.12em',
            textTransform: 'uppercase',
            color: '#302E2A',
            textDecoration: 'none',
            transition: 'color 150ms',
          }}
          onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6A6458')}
          onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#302E2A')}
        >
          ← Runs
        </Link>
        <p style={{ marginTop: 24, fontSize: 12, color: '#FF3A3A', letterSpacing: '0.04em' }}>
          run not found.
        </p>
      </div>
    );
  }

  return (
    <div className="animate-slide-up" style={{ maxWidth: 640 }}>
      {/* Breadcrumb */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 32 }}>
        <Link
          to="/runs"
          style={{
            fontSize: 10,
            letterSpacing: '0.12em',
            textTransform: 'uppercase',
            color: '#302E2A',
            textDecoration: 'none',
            transition: 'color 150ms',
          }}
          onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6A6458')}
          onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#302E2A')}
        >
          Runs
        </Link>
        <span style={{ color: '#191919', fontSize: 10 }}>/</span>
        <span style={{ fontSize: 10, letterSpacing: '0.1em', color: '#403C34' }}>
          #{run.id}
        </span>
        <StatusBadge status={run.status} />
      </div>

      {/* Fields — terminal log style */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: '20px 32px',
          padding: '24px 0',
          borderTop: '1px solid #191919',
          borderBottom: '1px solid #191919',
          marginBottom: 24,
        }}
      >
        <Field label="Repository">
          {run.owner}/{run.repo}
        </Field>
        <Field label="Issue">#{run.issue_number}</Field>
        <Field label="Branch">
          {run.branch ? (
            <span style={{ fontSize: 11, color: '#6A6458', letterSpacing: '0.02em' }}>
              {run.branch}
            </span>
          ) : (
            <span style={{ color: '#302E2A' }}>—</span>
          )}
        </Field>
        <Field label="Pull Request">
          {run.pr_url ? (
            <a
              href={run.pr_url}
              target="_blank"
              rel="noreferrer"
              style={{
                color: '#3A9EFF',
                textDecoration: 'none',
                fontSize: 12,
                letterSpacing: '0.03em',
                transition: 'color 150ms',
              }}
              onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6ABBFF')}
              onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#3A9EFF')}
            >
              → View PR
            </a>
          ) : (
            <span style={{ color: '#302E2A' }}>—</span>
          )}
        </Field>
        <Field label="Created">{fmt(run.created_at)}</Field>
        <Field label="Updated">{fmt(run.updated_at)}</Field>
      </div>

      {/* Error block */}
      {run.error && (
        <div
          style={{
            marginBottom: 24,
            padding: '16px',
            background: 'rgba(255, 58, 58, 0.04)',
            borderLeft: '2px solid #FF3A3A',
          }}
        >
          <span
            style={{
              display: 'block',
              fontSize: 10,
              letterSpacing: '0.2em',
              textTransform: 'uppercase',
              color: '#FF3A3A',
              marginBottom: 8,
            }}
          >
            Error
          </span>
          <pre
            style={{
              fontSize: 11,
              color: '#C45050',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              lineHeight: 1.7,
              margin: 0,
            }}
          >
            {run.error}
          </pre>
        </div>
      )}

      {/* Actions */}
      <div style={{ display: 'flex', gap: 12 }}>
        {(run.status === 'failed' || run.status === 'cancelled') && (
          <button
            onClick={() => retry.mutate()}
            disabled={retry.isPending}
            style={{
              padding: '8px 16px',
              background: '#3A9EFF',
              color: '#070707',
              fontSize: 10,
              letterSpacing: '0.15em',
              textTransform: 'uppercase',
              fontWeight: 600,
              border: 'none',
              cursor: retry.isPending ? 'not-allowed' : 'pointer',
              opacity: retry.isPending ? 0.5 : 1,
              transition: 'opacity 150ms',
            }}
            onMouseEnter={(e) => {
              if (!retry.isPending) (e.target as HTMLElement).style.opacity = '0.85';
            }}
            onMouseLeave={(e) => {
              if (!retry.isPending) (e.target as HTMLElement).style.opacity = '1';
            }}
          >
            {retry.isPending ? 'Retrying…' : 'Retry'}
          </button>
        )}
        {(run.status === 'pending' || run.status === 'in_progress') && (
          <button
            onClick={() => cancel.mutate()}
            disabled={cancel.isPending}
            style={{
              padding: '8px 16px',
              background: '#0C0C0C',
              color: '#403C34',
              fontSize: 10,
              letterSpacing: '0.15em',
              textTransform: 'uppercase',
              fontWeight: 600,
              border: '1px solid #191919',
              cursor: cancel.isPending ? 'not-allowed' : 'pointer',
              opacity: cancel.isPending ? 0.5 : 1,
              transition: 'opacity 150ms, color 150ms',
            }}
            onMouseEnter={(e) => {
              if (!cancel.isPending) (e.target as HTMLElement).style.color = '#C4C0AC';
            }}
            onMouseLeave={(e) => {
              if (!cancel.isPending) (e.target as HTMLElement).style.color = '#403C34';
            }}
          >
            {cancel.isPending ? 'Cancelling…' : 'Cancel'}
          </button>
        )}
      </div>
    </div>
  );
}
