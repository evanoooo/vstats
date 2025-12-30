interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{ value: number; name: string; color: string }>;
  label?: string;
  formatValue?: (v: number) => string;
  labelFormatter?: (label: string) => string;
  isLight?: boolean;
}

export function CustomTooltip({ 
  active, 
  payload, 
  label, 
  formatValue,
  labelFormatter,
  isLight,
}: CustomTooltipProps) {
  if (!active || !payload?.length) return null;
  
  return (
    <div className={`backdrop-blur-sm rounded-lg p-3 shadow-xl ${
      isLight 
        ? 'bg-white/95 border border-gray-200' 
        : 'bg-gray-900/95 border border-white/10'
    }`}>
      <p className={`text-xs mb-2 font-mono ${isLight ? 'text-gray-500' : 'text-gray-400'}`}>
        {labelFormatter ? labelFormatter(label || '') : label}
      </p>
      {payload.map((entry, index) => (
        <div key={index} className="flex items-center gap-2">
          <span 
            className="w-2 h-2 rounded-full" 
            style={{ backgroundColor: entry.color }}
          />
          <span className={`text-xs ${isLight ? 'text-gray-600' : 'text-gray-300'}`}>{entry.name}:</span>
          <span className={`text-sm font-mono font-semibold ${isLight ? 'text-gray-900' : 'text-white'}`}>
            {formatValue ? formatValue(entry.value) : entry.value}
          </span>
        </div>
      ))}
    </div>
  );
}

