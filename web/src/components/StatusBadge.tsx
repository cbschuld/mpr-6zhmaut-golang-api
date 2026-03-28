interface Props {
  state: string;
}

export function StatusBadge({ state }: Props) {
  const isReady = state === "READY" || state === "ready";
  return (
    <span className="flex items-center gap-1.5 text-xs font-medium">
      <span
        className={`w-2 h-2 rounded-full ${isReady ? "bg-on animate-pulse" : "bg-amber-500 animate-pulse"}`}
      />
      {state.toUpperCase()}
    </span>
  );
}
