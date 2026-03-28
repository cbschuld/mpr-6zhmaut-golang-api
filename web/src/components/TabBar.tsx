export type Tab = "zones" | "dashboard";

interface Props {
  active: Tab;
  onChange: (tab: Tab) => void;
}

export function TabBar({ active, onChange }: Props) {
  return (
    <div className="flex border-b border-gray-700">
      <TabButton label="Zones" active={active === "zones"} onClick={() => onChange("zones")} />
      <TabButton label="Dashboard" active={active === "dashboard"} onClick={() => onChange("dashboard")} />
    </div>
  );
}

function TabButton({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 py-2.5 text-sm font-medium transition-colors ${
        active
          ? "text-primary border-b-2 border-primary"
          : "text-gray-400 hover:text-gray-200"
      }`}
    >
      {label}
    </button>
  );
}
