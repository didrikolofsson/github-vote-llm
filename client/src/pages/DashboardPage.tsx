import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { listRuns, cancelRun, retryRun } from '../client/sdk.gen';
import type { Run } from '../client/types.gen';
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

const TH_STYLE: React.CSSProperties = {
  paddingBottom: 8,
  paddingRight: 24,
  fontSize: 10,
  letterSpacing: '0.15em',
  textTransform: 'uppercase',
  color: '#302E2A',
  fontWeight: 400,
  textAlign: 'left',
  borderBottom: '1px solid #191919',
  whiteSpace: 'nowrap',
};

export default function DashboardPage() {
  const qc = useQueryClient();

  const { data: runs, isLoading, error } = useQuery({
    queryKey: ['runs'],
    queryFn: () => listRuns().then((r) => r.data ?? []),
    refetchInterval: 15_000,
  });

  const cancel = useMutation({
    mutationFn: (id: number) => cancelRun({ path: { id } }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs'] }),
  });

  const retry = useMutation({
    mutationFn: (id: number) => retryRun({ path: { id } }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs'] }),
  });

  if (isLoading) {
    return (
      <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.08em' }}>
        Loading…
      </p>
    );
  }

  if (error) {
    return (
      <p style={{ fontSize: 12, color: '#FF3A3A', letterSpacing: '0.04em' }}>
        error: {error instanceof Error ? error.message : 'unknown'}
      </p>
    );
  }

  return (
    <div className="animate-slide-up">
      <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 12 }}>
          <span
            style={{
              fontSize: 10,
              letterSpacing: '0.25em',
              textTransform: 'uppercase',
              color: '#00E87A',
            }}
          >
            Runs
          </span>
          {runs && (
            <span style={{ fontSize: 10, color: '#302E2A', letterSpacing: '0.05em' }}>
              {runs.length} total
            </span>
          )}
        </div>
        <button
          onClick={() => qc.invalidateQueries({ queryKey: ['runs'] })}
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

      {runs?.length === 0 ? (
        <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.06em' }}>
          no runs yet.
        </p>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={TH_STYLE}>ID</th>
                <th style={TH_STYLE}>Repository</th>
                <th style={TH_STYLE}>Issue</th>
                <th style={TH_STYLE}>Status</th>
                <th style={TH_STYLE}>Branch</th>
                <th style={TH_STYLE}>Updated</th>
                <th style={{ ...TH_STYLE, paddingRight: 0 }}></th>
              </tr>
            </thead>
            <tbody>
              {runs?.map((run: Run, i: number) => (
                <tr
                  key={run.id}
                  style={{
                    borderBottom: '1px solid #111111',
                    animationDelay: `${i * 30}ms`,
                  }}
                  onMouseEnter={(e) =>
                    ((e.currentTarget as HTMLElement).style.background = '#0C0C0C')
                  }
                  onMouseLeave={(e) =>
                    ((e.currentTarget as HTMLElement).style.background = 'transparent')
                  }
                >
                  <td style={{ padding: '10px 24px 10px 0', color: '#403C34' }}>
                    <Link
                      to={`/runs/${run.id}`}
                      style={{
                        color: '#403C34',
                        textDecoration: 'none',
                        letterSpacing: '0.04em',
                        transition: 'color 150ms',
                      }}
                      onMouseEnter={(e) =>
                        ((e.target as HTMLElement).style.color = '#C4C0AC')
                      }
                      onMouseLeave={(e) =>
                        ((e.target as HTMLElement).style.color = '#403C34')
                      }
                    >
                      #{run.id}
                    </Link>
                  </td>
                  <td
                    style={{
                      padding: '10px 24px 10px 0',
                      color: '#C4C0AC',
                      letterSpacing: '0.02em',
                    }}
                  >
                    {run.owner}/{run.repo}
                  </td>
                  <td
                    style={{
                      padding: '10px 24px 10px 0',
                      color: '#403C34',
                      letterSpacing: '0.04em',
                    }}
                  >
                    #{run.issue_number}
                  </td>
                  <td style={{ padding: '10px 24px 10px 0' }}>
                    <StatusBadge status={run.status} />
                  </td>
                  <td
                    style={{
                      padding: '10px 24px 10px 0',
                      color: '#302E2A',
                      fontSize: 11,
                      letterSpacing: '0.02em',
                      maxWidth: 220,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {run.branch ?? '—'}
                  </td>
                  <td
                    style={{
                      padding: '10px 24px 10px 0',
                      color: '#302E2A',
                      fontSize: 11,
                      letterSpacing: '0.04em',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {relativeTime(run.updated_at)}
                  </td>
                  <td style={{ padding: '10px 0', textAlign: 'right' }}>
                    {(run.status === 'failed' || run.status === 'cancelled') && (
                      <button
                        onClick={() => retry.mutate(run.id)}
                        disabled={retry.isPending}
                        style={{
                          fontSize: 10,
                          letterSpacing: '0.12em',
                          textTransform: 'uppercase',
                          color: '#3A9EFF',
                          background: 'none',
                          border: 'none',
                          cursor: 'pointer',
                          opacity: retry.isPending ? 0.4 : 1,
                          transition: 'color 150ms',
                        }}
                        onMouseEnter={(e) =>
                          ((e.target as HTMLElement).style.color = '#6ABBFF')
                        }
                        onMouseLeave={(e) =>
                          ((e.target as HTMLElement).style.color = '#3A9EFF')
                        }
                      >
                        Retry
                      </button>
                    )}
                    {(run.status === 'pending' || run.status === 'in_progress') && (
                      <button
                        onClick={() => cancel.mutate(run.id)}
                        disabled={cancel.isPending}
                        style={{
                          fontSize: 10,
                          letterSpacing: '0.12em',
                          textTransform: 'uppercase',
                          color: '#302E2A',
                          background: 'none',
                          border: 'none',
                          cursor: 'pointer',
                          opacity: cancel.isPending ? 0.4 : 1,
                          transition: 'color 150ms',
                        }}
                        onMouseEnter={(e) =>
                          ((e.target as HTMLElement).style.color = '#6A6458')
                        }
                        onMouseLeave={(e) =>
                          ((e.target as HTMLElement).style.color = '#302E2A')
                        }
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
