import { useState, useEffect, useCallback } from "react";
import type { Zone } from "../api/types";
import { getZones, setAttribute } from "../api/client";

export function useZones(pollIntervalMs = 5000) {
  const [zones, setZones] = useState<Zone[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchZones = useCallback(async () => {
    try {
      const data = await getZones();
      setZones(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch zones");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchZones();
    const interval = setInterval(fetchZones, pollIntervalMs);
    return () => clearInterval(interval);
  }, [fetchZones, pollIntervalMs]);

  const setZoneAttribute = useCallback(
    async (zoneId: string, attr: string, value: string) => {
      // Optimistic update
      setZones((prev) =>
        prev.map((z) => (z.zone === zoneId ? { ...z, [attr]: value } : z))
      );
      try {
        await setAttribute(zoneId, attr, value);
      } catch (err) {
        // Revert on failure by re-fetching
        fetchZones();
        console.error("Failed to set attribute:", err);
      }
    },
    [fetchZones]
  );

  return { zones, loading, error, setZoneAttribute, refresh: fetchZones };
}
