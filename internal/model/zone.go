package model

// Zone represents the state of a single amplifier zone.
type Zone struct {
	Zone    string `json:"zone"`
	PA      string `json:"pa"`
	Power   string `json:"pr"`
	Mute    string `json:"mu"`
	DT      string `json:"dt"`
	Volume  string `json:"vo"`
	Treble  string `json:"tr"`
	Bass    string `json:"bs"`
	Balance string `json:"bl"`
	Channel string `json:"ch"`
	Keypad  string `json:"ls"`
}

// GetAttribute returns the value of a named attribute.
func (z Zone) GetAttribute(attr string) string {
	switch attr {
	case "pa":
		return z.PA
	case "pr":
		return z.Power
	case "mu":
		return z.Mute
	case "dt":
		return z.DT
	case "vo":
		return z.Volume
	case "tr":
		return z.Treble
	case "bs":
		return z.Bass
	case "bl":
		return z.Balance
	case "ch":
		return z.Channel
	case "ls":
		return z.Keypad
	default:
		return ""
	}
}

// SetAttribute returns a copy of the zone with the named attribute updated.
func (z Zone) SetAttribute(attr, value string) Zone {
	switch attr {
	case "pa":
		z.PA = value
	case "pr":
		z.Power = value
	case "mu":
		z.Mute = value
	case "dt":
		z.DT = value
	case "vo":
		z.Volume = value
	case "tr":
		z.Treble = value
	case "bs":
		z.Bass = value
	case "bl":
		z.Balance = value
	case "ch":
		z.Channel = value
	case "ls":
		z.Keypad = value
	}
	return z
}
