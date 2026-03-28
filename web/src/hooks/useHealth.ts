import { useState, useEffect } from "react";
import type { HealthStatus } from "../api/types";
import { getHealth } from "../api/client";

export function useHealth(pollIntervalMs = 10000) {
  const [health, setHealth] = useState<HealthStatus | null>(null);

  useEffect(() => {
    const fetchHealth = async () => {
      try {
        const data = await getHealth();
        setHealth(data);
      } catch {
        // Health endpoint unavailable
      }
    };
    fetchHealth();
    const interval = setInterval(fetchHealth, pollIntervalMs);
    return () => clearInterval(interval);
  }, [pollIntervalMs]);

  return health;
}
