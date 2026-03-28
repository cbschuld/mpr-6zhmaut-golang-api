import type { Zone, GroupTag } from "../api/types";
import { GroupControl } from "./GroupControl";

interface Props {
  zones: Zone[];
  labels: Record<string, string>;
  sources: { ch: string; name: string }[];
  getZonesForTag: (tag: GroupTag) => string[];
  onSetAttribute: (zoneId: string, attr: string, value: string) => void;
}

export function Dashboard({ zones, labels, sources, getZonesForTag, onSetAttribute }: Props) {
  const interiorIds = getZonesForTag("interior");
  const exteriorIds = getZonesForTag("exterior");
  const livingIds = getZonesForTag("living");

  const filterZones = (ids: string[]) => zones.filter((z) => ids.includes(z.zone));

  return (
    <div className="p-4 space-y-3">
      <GroupControl
        title="All Zones"
        zones={zones}
        zoneLabels={labels}
        sources={sources}
        onSetAttribute={onSetAttribute}
      />
      <div className="border-t border-gray-700 pt-3 space-y-3">
        <GroupControl
          title="Interior"
          zones={filterZones(interiorIds)}
          zoneLabels={labels}
          sources={sources}
          onSetAttribute={onSetAttribute}
        />
        <GroupControl
          title="Exterior"
          zones={filterZones(exteriorIds)}
          zoneLabels={labels}
          sources={sources}
          onSetAttribute={onSetAttribute}
        />
        <GroupControl
          title="Living Areas"
          zones={filterZones(livingIds)}
          zoneLabels={labels}
          sources={sources}
          onSetAttribute={onSetAttribute}
        />
      </div>
    </div>
  );
}
