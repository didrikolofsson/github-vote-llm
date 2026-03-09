type Status = 'pending' | 'in_progress' | 'success' | 'failed' | 'cancelled';

const STATUS_MAP: Record<Status, { color: string; label: string; pulse?: boolean }> = {
  pending:     { color: '#ECA030', label: 'pending' },
  in_progress: { color: '#00E87A', label: 'in progress', pulse: true },
  success:     { color: '#00E87A', label: 'success' },
  failed:      { color: '#FF3A3A', label: 'failed' },
  cancelled:   { color: '#302E2A', label: 'cancelled' },
};

export default function StatusBadge({ status }: { status: string }) {
  const cfg = STATUS_MAP[status as Status] ?? { color: '#302E2A', label: status };

  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span
        className={cfg.pulse ? 'animate-pulse-dot' : undefined}
        style={{
          width: 5,
          height: 5,
          borderRadius: '50%',
          background: cfg.color,
          flexShrink: 0,
          boxShadow: cfg.pulse ? `0 0 6px ${cfg.color}` : undefined,
        }}
      />
      <span
        style={{
          color: cfg.color,
          fontSize: 11,
          letterSpacing: '0.06em',
          textTransform: 'uppercase',
        }}
      >
        {cfg.label}
      </span>
    </span>
  );
}
