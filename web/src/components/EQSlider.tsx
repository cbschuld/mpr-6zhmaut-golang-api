interface Props {
  label: string;
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
}

export function EQSlider({ label, value, onChange, min = 0, max = 14 }: Props) {
  return (
    <div className="flex items-center gap-3">
      <span className="text-xs text-gray-400 w-16">{label}</span>
      <input
        type="range"
        min={min}
        max={max}
        value={value}
        onChange={(e) => onChange(parseInt(e.target.value))}
        className="flex-1"
      />
      <span className="text-xs font-mono w-4 text-right text-gray-400">{value}</span>
    </div>
  );
}
