type Status = 'pending' | 'in_progress' | 'success' | 'failed' | 'cancelled';

const STATUS_MAP: Record<Status, { colorClass: string; label: string; pulse?: boolean }> = {
  pending: { colorClass: 'text-amber-500', label: 'pending' },
  in_progress: { colorClass: 'text-emerald-400', label: 'in progress', pulse: true },
  success: { colorClass: 'text-emerald-400', label: 'success' },
  failed: { colorClass: 'text-red-500', label: 'failed' },
  cancelled: { colorClass: 'text-gray-500', label: 'cancelled' },
};

export default function StatusBadge({ status }: { status: string }) {
  const cfg = STATUS_MAP[status as Status] ?? { colorClass: 'text-gray-500', label: status };

  return (
    <span className={`inline-flex items-center gap-1.5 ${cfg.colorClass}`}>
      <span
        className={`w-1.5 h-1.5 rounded-full flex-shrink-0 bg-current ${cfg.pulse ? 'animate-pulse-dot shadow-[0_0_6px_rgb(52_211_153)]' : ''}`}
      />
      <span className="text-[11px] tracking-[0.06em] uppercase">{cfg.label}</span>
    </span>
  );
}
