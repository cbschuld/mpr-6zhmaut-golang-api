package amp

import (
	"sync"
	"time"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/model"
)

// ZoneCache is a thread-safe in-memory cache of zone states.
type ZoneCache struct {
	mu         sync.RWMutex
	zones      map[string]model.Zone
	lastUpdate time.Time
}

// NewZoneCache creates an empty zone cache.
func NewZoneCache() *ZoneCache {
	return &ZoneCache{
		zones: make(map[string]model.Zone),
	}
}

// Get returns a single zone by ID, or false if not found.
func (c *ZoneCache) Get(zoneID string) (model.Zone, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	z, ok := c.zones[zoneID]
	return z, ok
}

// GetAll returns a copy of all cached zones as a slice.
func (c *ZoneCache) GetAll() []model.Zone {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]model.Zone, 0, len(c.zones))
	for _, z := range c.zones {
		result = append(result, z)
	}
	return result
}

// Update replaces the cache with fresh zone data from a poll cycle.
// Returns a list of changes detected (for logging keypad changes).
func (c *ZoneCache) Update(zones []model.Zone) []ZoneChange {
	c.mu.Lock()
	defer c.mu.Unlock()

	var changes []ZoneChange
	for _, z := range zones {
		if old, ok := c.zones[z.Zone]; ok {
			changes = append(changes, diffZone(old, z)...)
		}
		c.zones[z.Zone] = z
	}
	c.lastUpdate = time.Now()
	return changes
}

// OptimisticSet updates a single attribute in the cache immediately.
func (c *ZoneCache) OptimisticSet(zoneID, attr, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if z, ok := c.zones[zoneID]; ok {
		c.zones[zoneID] = z.SetAttribute(attr, value)
	}
}

// Age returns how long since the cache was last updated.
func (c *ZoneCache) Age() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastUpdate.IsZero() {
		return 0
	}
	return time.Since(c.lastUpdate)
}

// LastUpdate returns the time of the last cache update.
func (c *ZoneCache) LastUpdate() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdate
}

// Count returns the number of zones in the cache.
func (c *ZoneCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.zones)
}

// ZoneChange represents a detected change in a zone attribute.
type ZoneChange struct {
	ZoneID   string
	Attr     string
	OldValue string
	NewValue string
}

func diffZone(old, new model.Zone) []ZoneChange {
	var changes []ZoneChange
	attrs := []struct {
		name   string
		oldVal string
		newVal string
	}{
		{"pa", old.PA, new.PA},
		{"pr", old.Power, new.Power},
		{"mu", old.Mute, new.Mute},
		{"dt", old.DT, new.DT},
		{"vo", old.Volume, new.Volume},
		{"tr", old.Treble, new.Treble},
		{"bs", old.Bass, new.Bass},
		{"bl", old.Balance, new.Balance},
		{"ch", old.Channel, new.Channel},
		{"ls", old.Keypad, new.Keypad},
	}
	for _, a := range attrs {
		if a.oldVal != a.newVal {
			changes = append(changes, ZoneChange{
				ZoneID:   new.Zone,
				Attr:     a.name,
				OldValue: a.oldVal,
				NewValue: a.newVal,
			})
		}
	}
	return changes
}
