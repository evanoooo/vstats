interface StatCardProps {
  label: string;
  value: string;
  subValue?: string;
  color?: 'emerald' | 'blue' | 'purple' | 'amber' | 'gray';
}

const colorClasses: Record<string, string> = {
  emerald: 'from-emerald-500/20 border-emerald-500/30',
  blue: 'from-blue-500/20 border-blue-500/30',
  purple: 'from-purple-500/20 border-purple-500/30',
  amber: 'from-amber-500/20 border-amber-500/30',
  gray: 'from-white/5 border-white/10',
};

export function StatCard({ label, value, subValue, color = 'gray' }: StatCardProps) {
  return (
    <div className={`nezha-card p-4 bg-gradient-to-br ${colorClasses[color]} to-transparent`}>
      <div className="text-[10px] uppercase tracking-wider text-gray-400 mb-1">{label}</div>
      <div className="text-xl font-bold text-white font-mono">{value}</div>
      {subValue && <div className="text-xs text-gray-500 mt-1">{subValue}</div>}
    </div>
  );
}

