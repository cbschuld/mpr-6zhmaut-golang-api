import type { ZoneSettings, GroupTag } from "../api/types";

interface Props {
  settings: ZoneSettings;
  onUpdate: (updates: Partial<ZoneSettings>) => void;
  onClose: () => void;
}

const ALL_ZONES = ["11", "12", "13", "14", "15", "16", "21", "22", "23", "24", "25", "26"];
const TAG_OPTIONS: { value: GroupTag; label: string }[] = [
  { value: "interior", label: "Interior" },
  { value: "exterior", label: "Exterior" },
  { value: "living", label: "Living Area" },
];

export function SettingsPage({ settings, onUpdate, onClose }: Props) {
  const updateLabel = (zoneId: string, label: string) => {
    onUpdate({ labels: { ...settings.labels, [zoneId]: label } });
  };

  const updateSource = (index: number, name: string) => {
    const sources = [...settings.sources];
    sources[index] = { ...sources[index], name };
    onUpdate({ sources });
  };

  const toggleTag = (zoneId: string, tag: GroupTag) => {
    const current = settings.tags[zoneId] || [];
    const next = current.includes(tag)
      ? current.filter((t) => t !== tag)
      : [...current, tag];
    onUpdate({ tags: { ...settings.tags, [zoneId]: next } });
  };

  return (
    <div className="fixed inset-0 bg-surface z-50 overflow-y-auto">
      <div className="max-w-lg mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700 sticky top-0 bg-surface">
          <h2 className="text-lg font-semibold text-white">Settings</h2>
          <button
            onClick={onClose}
            className="text-primary text-sm font-medium"
          >
            Done
          </button>
        </div>

        <div className="p-4 space-y-6">
          {/* Zone Labels */}
          <section>
            <h3 className="text-sm font-medium text-gray-300 mb-3">Zone Names</h3>
            <div className="space-y-2">
              {ALL_ZONES.map((id) => (
                <div key={id} className="flex items-center gap-3">
                  <span className="text-xs text-gray-500 w-8 font-mono">{id}</span>
                  <input
                    type="text"
                    value={settings.labels[id] || ""}
                    onChange={(e) => updateLabel(id, e.target.value)}
                    placeholder={`Zone ${id}`}
                    className="flex-1 bg-surface-light border border-gray-600 rounded px-3 py-1.5 text-sm text-white outline-none focus:border-primary"
                  />
                </div>
              ))}
            </div>
          </section>

          {/* Zone Tags */}
          <section>
            <h3 className="text-sm font-medium text-gray-300 mb-3">Zone Groups</h3>
            <div className="space-y-2">
              {ALL_ZONES.map((id) => (
                <div key={id} className="flex items-center gap-3">
                  <span className="text-xs text-gray-400 w-20 truncate">
                    {settings.labels[id] || `Zone ${id}`}
                  </span>
                  <div className="flex gap-1.5">
                    {TAG_OPTIONS.map((tag) => {
                      const active = (settings.tags[id] || []).includes(tag.value);
                      return (
                        <button
                          key={tag.value}
                          onClick={() => toggleTag(id, tag.value)}
                          className={`text-[10px] px-2 py-1 rounded-full border transition-colors ${
                            active
                              ? "bg-primary/20 border-primary text-primary"
                              : "border-gray-600 text-gray-500"
                          }`}
                        >
                          {tag.label}
                        </button>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </section>

          {/* Source Names */}
          <section>
            <h3 className="text-sm font-medium text-gray-300 mb-3">Source Names</h3>
            <div className="space-y-2">
              {settings.sources.map((src, i) => (
                <div key={src.ch} className="flex items-center gap-3">
                  <span className="text-xs text-gray-500 w-8 font-mono">{src.ch}</span>
                  <input
                    type="text"
                    value={src.name}
                    onChange={(e) => updateSource(i, e.target.value)}
                    placeholder={`Source ${parseInt(src.ch)}`}
                    className="flex-1 bg-surface-light border border-gray-600 rounded px-3 py-1.5 text-sm text-white outline-none focus:border-primary"
                  />
                </div>
              ))}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
