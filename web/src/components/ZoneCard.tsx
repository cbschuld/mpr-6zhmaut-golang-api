import { useState, useCallback } from "react";
import type { Zone } from "../api/types";
import { PowerToggle } from "./PowerToggle";
import { VolumeSlider } from "./VolumeSlider";
import { EQSlider } from "./EQSlider";
import { SourceSelector } from "./SourceSelector";
import { useDebounce } from "../hooks/useDebounce";

interface Props {
  zone: Zone;
  label: string;
  sources: { ch: string; name: string }[];
  onSetAttribute: (zoneId: string, attr: string, value: string) => void;
}

export function ZoneCard({ zone, label, sources, onSetAttribute }: Props) {
  const [expanded, setExpanded] = useState(false);
  const isOn = zone.pr === "01";

  const debouncedSet = useDebounce(
    useCallback(
      (attr: string, value: string) => onSetAttribute(zone.zone, attr, value),
      [zone.zone, onSetAttribute]
    ),
    300
  );

  return (
    <div
      className={`rounded-lg border transition-colors ${
        isOn ? "bg-surface-light border-gray-600" : "bg-surface border-gray-700 opacity-60"
      }`}
    >
      <div className="p-4">
        {/* Top row: label + power */}
        <div className="flex items-center justify-between mb-3">
          <span className="text-sm font-medium text-white">{label}</span>
          <PowerToggle
            on={isOn}
            onChange={(on) => onSetAttribute(zone.zone, "pr", on ? "01" : "00")}
          />
        </div>

        {/* Volume + source (visible when on) */}
        {isOn && (
          <div className="space-y-2.5">
            <VolumeSlider
              value={parseInt(zone.vo)}
              onChange={(v) => debouncedSet("vo", String(v).padStart(2, "0"))}
            />
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-400">Source</span>
                <SourceSelector
                  value={zone.ch}
                  sources={sources}
                  onChange={(ch) => onSetAttribute(zone.zone, "ch", ch)}
                />
              </div>
              <button
                onClick={() => setExpanded(!expanded)}
                className={`text-xs px-2 py-1 rounded transition-colors ${
                  expanded
                    ? "bg-primary/20 text-primary"
                    : "text-gray-400 hover:text-gray-200"
                }`}
              >
                EQ {expanded ? "▲" : "▼"}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* EQ controls (expanded) */}
      {isOn && expanded && (
        <div className="px-4 pb-4 pt-1 space-y-2 border-t border-gray-700">
          <EQSlider
            label="Treble"
            value={parseInt(zone.tr)}
            onChange={(v) => debouncedSet("tr", String(v).padStart(2, "0"))}
          />
          <EQSlider
            label="Bass"
            value={parseInt(zone.bs)}
            onChange={(v) => debouncedSet("bs", String(v).padStart(2, "0"))}
          />
          <EQSlider
            label="Balance"
            value={parseInt(zone.bl)}
            max={20}
            onChange={(v) => debouncedSet("bl", String(v).padStart(2, "0"))}
          />
          <button
            onClick={() => {
              onSetAttribute(zone.zone, "tr", "07");
              onSetAttribute(zone.zone, "bs", "07");
              onSetAttribute(zone.zone, "bl", "10");
            }}
            className="w-full text-xs py-1.5 rounded border border-gray-600 text-gray-400 hover:text-white hover:border-gray-400 transition-colors mt-1"
          >
            Reset EQ
          </button>
        </div>
      )}
    </div>
  );
}
