import type { Zone, HealthStatus } from "./types";

export async function getZones(): Promise<Zone[]> {
  const res = await fetch("/api/zones");
  if (!res.ok) throw new Error(`GET /api/zones: ${res.status}`);
  return res.json();
}

export async function setAttribute(
  zoneId: string,
  attr: string,
  value: string
): Promise<Zone> {
  const res = await fetch(`/api/zones/${zoneId}/${attr}`, {
    method: "POST",
    body: value,
  });
  if (!res.ok) throw new Error(`POST /api/zones/${zoneId}/${attr}: ${res.status}`);
  return res.json();
}

export async function getHealth(): Promise<HealthStatus> {
  const res = await fetch("/api/health");
  if (!res.ok) throw new Error(`GET /api/health: ${res.status}`);
  return res.json();
}
