import { useCallback } from "react";
import type { Zone } from "../api/types";
import { PowerToggle } from "./PowerToggle";
import { VolumeSlider } from "./VolumeSlider";
import { SourceSelector } from "./SourceSelector";
import { useDebounce } from "../hooks/useDebounce";

interface Props {
  title: string;
  zones: Zone[];
  zoneLabels: Record<string, string>;
  sources: { ch: string; name: string }[];
  onSetAttribute: (zoneId: string, attr: string, value: string) => void;
}

export function GroupControl({ title, zones, zoneLabels, sources, onSetAttribute }: Props) {
  const poweredOn = zones.filter((z) => z.pr === "01");
  const anyOn = poweredOn.length > 0;
  const avgVolume =
    poweredOn.length > 0
      ? Math.round(poweredOn.reduce((sum, z) => sum + parseInt(z.vo), 0) / poweredOn.length)
      : 0;

  const setAll = useCallback(
    (attr: string, value: string) => {
      zones.forEach((z) => onSetAttribute(z.zone, attr, value));
    },
    [zones, onSetAttribute]
  );

  const debouncedSetAll = useDebounce(
    useCallback(
      (attr: string, value: string) => setAll(attr, value),
      [setAll]
    ),
    300
  );

  return (
    <div className="rounded-lg bg-surface-light border border-gray-600 p-4">
      <div className="flex items-center justify-between mb-3">
        <div>
          <h3 className="text-sm font-medium text-white">{title}</h3>
          <span className="text-xs text-gray-400">{zones.length} zones</span>
        </div>
        <PowerToggle
          on={anyOn}
          onChange={(on) => setAll("pr", on ? "01" : "00")}
        />
      </div>

      {anyOn && (
        <div className="space-y-2.5">
          <VolumeSlider
            value={avgVolume}
            onChange={(v) => debouncedSetAll("vo", String(v).padStart(2, "0"))}
          />
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400">Source</span>
            <SourceSelector
              value={poweredOn[0]?.ch || "01"}
              sources={sources}
              onChange={(ch) => setAll("ch", ch)}
            />
          </div>
        </div>
      )}

      {/* Zone indicators */}
      <div className="flex flex-wrap gap-1.5 mt-3">
        {zones.map((z) => (
          <span
            key={z.zone}
            className={`text-[10px] px-1.5 py-0.5 rounded ${
              z.pr === "01" ? "bg-on/20 text-on" : "bg-gray-700 text-gray-500"
            }`}
          >
            {zoneLabels[z.zone] || z.zone}
          </span>
        ))}
      </div>
    </div>
  );
}
