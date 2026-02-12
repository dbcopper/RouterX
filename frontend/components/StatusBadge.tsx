interface StatusBadgeProps {
  status: 'ok' | 'fail' | 'unknown' | 'active' | 'inactive';
  label?: string;
}

const colors: Record<string, string> = {
  ok: 'bg-emerald-400',
  active: 'bg-emerald-400',
  fail: 'bg-red-400',
  unknown: 'bg-gray-300',
  inactive: 'bg-gray-300',
};

export default function StatusBadge({ status, label }: StatusBadgeProps) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs">
      <span className={`w-2 h-2 rounded-full ${colors[status] || 'bg-gray-300'}`} />
      {label && <span className="text-black/60">{label}</span>}
    </span>
  );
}
