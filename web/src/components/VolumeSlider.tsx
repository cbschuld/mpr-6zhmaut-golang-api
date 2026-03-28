interface Props {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
}

export function VolumeSlider({ value, onChange, min = 0, max = 38 }: Props) {
  return (
    <div className="flex items-center gap-3 flex-1">
      <svg className="w-4 h-4 text-gray-400 shrink-0" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z" />
      </svg>
      <input
        type="range"
        min={min}
        max={max}
        value={value}
        onChange={(e) => onChange(parseInt(e.target.value))}
        className="flex-1"
      />
      <span className="text-sm font-mono w-6 text-right text-gray-300">{value}</span>
    </div>
  );
}
