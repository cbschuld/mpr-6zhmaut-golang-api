import { useState } from "react";
import { Header } from "./components/Header";
import { TabBar, type Tab } from "./components/TabBar";
import { ZoneCard } from "./components/ZoneCard";
import { Dashboard } from "./components/Dashboard";
import { SettingsPage } from "./components/SettingsPage";
import { useZones } from "./hooks/useZones";
import { useHealth } from "./hooks/useHealth";
import { useSettings } from "./hooks/useSettings";

function App() {
  const [tab, setTab] = useState<Tab>("zones");
  const [showSettings, setShowSettings] = useState(false);
  const { zones, loading, error, setZoneAttribute } = useZones();
  const health = useHealth();
  const { settings, updateSettings, getLabel, getZonesForTag } = useSettings();

  const state = health?.state_machine?.state || "CONNECTING";

  if (showSettings) {
    return (
      <SettingsPage
        settings={settings}
        onUpdate={updateSettings}
        onClose={() => setShowSettings(false)}
      />
    );
  }

  return (
    <div className="max-w-lg mx-auto min-h-screen flex flex-col">
      <Header state={state} onSettingsClick={() => setShowSettings(true)} />
      <TabBar active={tab} onChange={setTab} />

      <main className="flex-1">
        {loading && zones.length === 0 && (
          <div className="p-8 text-center text-gray-400">Connecting to amplifier...</div>
        )}

        {error && zones.length === 0 && (
          <div className="p-8 text-center text-red-400">{error}</div>
        )}

        {tab === "zones" && zones.length > 0 && (
          <div className="p-4 space-y-3">
            {zones.map((zone) => (
              <ZoneCard
                key={zone.zone}
                zone={zone}
                label={getLabel(zone.zone)}
                sources={settings.sources}
                onSetAttribute={setZoneAttribute}
              />
            ))}
          </div>
        )}

        {tab === "dashboard" && zones.length > 0 && (
          <Dashboard
            zones={zones}
            labels={settings.labels}
            sources={settings.sources}
            getZonesForTag={getZonesForTag}
            onSetAttribute={setZoneAttribute}
          />
        )}
      </main>

      {/* Footer status */}
      {health && (
        <footer className="px-4 py-2 border-t border-gray-700 text-xs text-gray-500 flex justify-between">
          <span>
            Amp 1-{health.amps.count}: {health.serial.current_baud_rate} baud
          </span>
          <span>
            Cache: {Math.round(health.cache.cache_age_ms / 1000)}s ago
          </span>
        </footer>
      )}
    </div>
  );
}

export default App;
