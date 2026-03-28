import { useState, useEffect, useCallback, useRef } from "react";
import type { Zone } from "../api/types";
import { getZones, setAttribute } from "../api/client";

// How long to protect an optimistic value from being overwritten by polls
const OPTIMISTIC_GUARD_MS = 8000;

interface PendingUpdate {
  attr: string;
  value: string;
  timestamp: number;
}

export function useZones(pollIntervalMs = 5000) {
  const [zones, setZones] = useState<Zone[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const pendingUpdates = useRef<Map<string, PendingUpdate[]>>(new Map());

  const mergeZones = useCallback((incoming: Zone[]) => {
    const now = Date.now();

    setZones((prev) => {
      // Build a map of existing zones for merging
      const existing = new Map(prev.map((z) => [z.zone, z]));

      // Merge incoming data into existing state
      for (const z of incoming) {
        const pending = pendingUpdates.current.get(z.zone) || [];
        // Filter out expired pending updates
        const activePending = pending.filter(
          (p) => now - p.timestamp < OPTIMISTIC_GUARD_MS
        );
        pendingUpdates.current.set(z.zone, activePending);

        // Apply incoming data but preserve optimistically-set attributes
        const merged = { ...z };
        for (const p of activePending) {
          (merged as Record<string, string>)[p.attr] = p.value;
        }
        existing.set(z.zone, merged);
      }

      // Return sorted array -- never remove zones that weren't in this response
      return Array.from(existing.values()).sort((a, b) =>
        a.zone.localeCompare(b.zone)
      );
    });
  }, []);

  const fetchZones = useCallback(async () => {
    try {
      const data = await getZones();
      mergeZones(data);
      setError(null);
    } catch (err) {
      // Don't clear zones on error -- keep showing last known state
      setError(err instanceof Error ? err.message : "Failed to fetch zones");
    } finally {
      setLoading(false);
    }
  }, [mergeZones]);

  useEffect(() => {
    fetchZones();
    const interval = setInterval(fetchZones, pollIntervalMs);
    return () => clearInterval(interval);
  }, [fetchZones, pollIntervalMs]);

  const setZoneAttribute = useCallback(
    async (zoneId: string, attr: string, value: string) => {
      // Track this as a pending optimistic update
      const pending = pendingUpdates.current.get(zoneId) || [];
      // Replace any existing pending update for the same attribute
      const filtered = pending.filter((p) => p.attr !== attr);
      filtered.push({ attr, value, timestamp: Date.now() });
      pendingUpdates.current.set(zoneId, filtered);

      // Optimistic update
      setZones((prev) =>
        prev.map((z) => (z.zone === zoneId ? { ...z, [attr]: value } : z))
      );

      try {
        await setAttribute(zoneId, attr, value);
      } catch (err) {
        // Clear the pending guard so the next poll can correct the value
        const current = pendingUpdates.current.get(zoneId) || [];
        pendingUpdates.current.set(
          zoneId,
          current.filter((p) => p.attr !== attr)
        );
        console.error("Failed to set attribute:", err);
      }
    },
    []
  );

  return { zones, loading, error, setZoneAttribute, refresh: fetchZones };
}
