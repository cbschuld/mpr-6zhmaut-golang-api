export interface Zone {
  zone: string;
  pa: string;
  pr: string; // power: "00" off, "01" on
  mu: string; // mute: "00" off, "01" on
  dt: string;
  vo: string; // volume
  tr: string; // treble
  bs: string; // bass
  bl: string; // balance
  ch: string; // channel/source
  ls: string; // keypad status
}

export interface HealthStatus {
  status: string;
  uptime_seconds: number;
  serial: {
    device: string;
    current_baud_rate: number;
    target_baud_rate: number;
  };
  state_machine: {
    state: string;
    last_transition: string;
    time_in_state_seconds: number;
  };
  cache: {
    zone_count: number;
    last_poll: string;
    cache_age_ms: number;
    poll_interval: string;
  };
  queue: {
    pending_commands: number;
    total_commands_sent: number;
    total_timeouts: number;
    total_errors: number;
  };
  recovery: {
    total_recoveries: number;
    last_recovery: string;
    last_recovery_reason: string;
    last_recovery_duration_ms: number;
    consecutive_errors: number;
  };
  amps: {
    count: number;
  };
}

export type GroupTag = "interior" | "exterior" | "living";

export interface ZoneSettings {
  labels: Record<string, string>;
  sources: { ch: string; name: string }[];
  tags: Record<string, GroupTag[]>;
}
