import { useState, useCallback } from "react";
import type { ZoneSettings, GroupTag } from "../api/types";

const STORAGE_KEY = "mpr-settings";

const DEFAULT_SETTINGS: ZoneSettings = {
  labels: {
    "11": "Brock",
    "12": "Bode",
    "13": "Dining",
    "14": "Kitchen",
    "15": "Courtyard",
    "16": "Den",
    "21": "Great Room",
    "22": "Guest Room",
    "23": "Master Bed",
    "24": "Master Bath",
    "25": "Back Patio",
    "26": "Casita",
  },
  sources: [
    { ch: "01", name: "Source 1" },
    { ch: "02", name: "Source 2" },
    { ch: "03", name: "moOde" },
    { ch: "04", name: "Source 4" },
    { ch: "05", name: "Source 5" },
    { ch: "06", name: "Source 6" },
  ],
  tags: {
    "11": ["interior"],
    "12": ["interior"],
    "13": ["interior", "living"],
    "14": ["interior", "living"],
    "15": ["exterior"],
    "16": ["interior", "living"],
    "21": ["interior", "living"],
    "22": ["interior"],
    "23": ["interior"],
    "24": ["interior"],
    "25": ["exterior"],
    "26": ["exterior"],
  },
};

function loadSettings(): ZoneSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      return { ...DEFAULT_SETTINGS, ...parsed };
    }
  } catch {
    // ignore
  }
  return DEFAULT_SETTINGS;
}

function saveSettings(settings: ZoneSettings) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
}

export function useSettings() {
  const [settings, setSettingsState] = useState<ZoneSettings>(loadSettings);

  const updateSettings = useCallback((updates: Partial<ZoneSettings>) => {
    setSettingsState((prev) => {
      const next = { ...prev, ...updates };
      saveSettings(next);
      return next;
    });
  }, []);

  const getLabel = useCallback(
    (zoneId: string) => settings.labels[zoneId] || `Zone ${zoneId}`,
    [settings.labels]
  );

  const getSourceName = useCallback(
    (ch: string) => {
      const src = settings.sources.find((s) => s.ch === ch);
      return src?.name || `Source ${parseInt(ch)}`;
    },
    [settings.sources]
  );

  const getZonesForTag = useCallback(
    (tag: GroupTag): string[] => {
      return Object.entries(settings.tags)
        .filter(([, tags]) => tags.includes(tag))
        .map(([id]) => id);
    },
    [settings.tags]
  );

  return { settings, updateSettings, getLabel, getSourceName, getZonesForTag };
}
