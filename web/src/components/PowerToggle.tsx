interface Props {
  on: boolean;
  onChange: (on: boolean) => void;
}

export function PowerToggle({ on, onChange }: Props) {
  return (
    <button
      onClick={() => onChange(!on)}
      className={`relative w-12 h-7 rounded-full transition-colors ${on ? "bg-on" : "bg-off"}`}
      aria-label={on ? "Turn off" : "Turn on"}
    >
      <span
        className={`absolute top-0.5 left-0.5 w-6 h-6 rounded-full bg-white transition-transform shadow ${on ? "translate-x-5" : ""}`}
      />
    </button>
  );
}
