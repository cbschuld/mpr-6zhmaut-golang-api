package serial

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/model"
)

var zoneResponseRe = regexp.MustCompile(`#>(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})`)

const ErrorResponse = "Command Error."

// QueryCommand returns the command to query all zones on an amplifier.
// ampID is 1-based (1, 2, or 3).
func QueryCommand(ampID int) string {
	return fmt.Sprintf("?%d0\r", ampID)
}

// ControlCommand returns the command to set a zone attribute.
func ControlCommand(zoneID, attr, value string) string {
	return fmt.Sprintf("<%s%s%s\r", zoneID, attr, value)
}

// BaudRateCommand returns the command to change the amplifier's baud rate.
func BaudRateCommand(baudRate int) string {
	return fmt.Sprintf("<%d\r", baudRate)
}

// ParseZoneResponse attempts to parse a line as a zone status response.
// Returns nil if the line doesn't match the expected format.
func ParseZoneResponse(line string) *model.Zone {
	m := zoneResponseRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	return &model.Zone{
		Zone:    m[1],
		PA:      m[2],
		Power:   m[3],
		Mute:    m[4],
		DT:      m[5],
		Volume:  m[6],
		Treble:  m[7],
		Bass:    m[8],
		Balance: m[9],
		Channel: m[10],
		Keypad:  m[11],
	}
}

// IsErrorResponse returns true if the line is a command error response.
func IsErrorResponse(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), ErrorResponse)
}

// ResolveAttribute maps user-friendly attribute names to protocol codes.
func ResolveAttribute(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "pa":
		return "pa", true
	case "pr", "power":
		return "pr", true
	case "mu", "mute":
		return "mu", true
	case "dt":
		return "dt", true
	case "vo", "volume":
		return "vo", true
	case "tr", "treble":
		return "tr", true
	case "bs", "bass":
		return "bs", true
	case "bl", "balance":
		return "bl", true
	case "ch", "channel", "source":
		return "ch", true
	case "ls", "keypad":
		return "ls", true
	default:
		return "", false
	}
}

// AmpIDForZone returns the amplifier ID (1-3) for a given zone ID string.
func AmpIDForZone(zoneID string) int {
	if len(zoneID) < 1 {
		return 0
	}
	return int(zoneID[0] - '0')
}

// ValidZoneID checks if a zone ID string is valid for the given amp count.
func ValidZoneID(zoneID string, ampCount int) bool {
	if len(zoneID) != 2 {
		return false
	}
	ampDigit := int(zoneID[0] - '0')
	zoneDigit := int(zoneID[1] - '0')
	if ampDigit < 1 || ampDigit > ampCount {
		return false
	}
	if zoneDigit < 1 || zoneDigit > 6 {
		return false
	}
	return true
}

// BaudRateSteps returns the ordered list of baud rates to step through
// from current to target. Returns nil if current >= target or current is not valid.
func BaudRateSteps(current, target int) []int {
	rates := []int{9600, 19200, 38400, 57600, 115200, 230400}
	var steps []int
	started := false
	for _, r := range rates {
		if r == current {
			started = true
			continue
		}
		if started {
			steps = append(steps, r)
			if r == target {
				break
			}
		}
	}
	return steps
}
