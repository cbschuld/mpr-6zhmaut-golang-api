interface Source {
  ch: string;
  name: string;
}

interface Props {
  value: string;
  sources: Source[];
  onChange: (ch: string) => void;
}

export function SourceSelector({ value, sources, onChange }: Props) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="bg-surface-lighter text-sm text-gray-200 rounded px-2 py-1 border border-gray-600 outline-none focus:border-primary"
    >
      {sources.map((s) => (
        <option key={s.ch} value={s.ch}>
          {s.name}
        </option>
      ))}
    </select>
  );
}
