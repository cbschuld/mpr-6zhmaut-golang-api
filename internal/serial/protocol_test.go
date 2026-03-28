package serial

import (
	"testing"
)

func TestParseZoneResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZone string
		wantNil  bool
	}{
		{
			name:     "valid zone 11",
			input:    "#>1100010100150710100601",
			wantZone: "11",
		},
		{
			name:     "valid zone 26",
			input:    "#>2600000000000707100100",
			wantZone: "26",
		},
		{
			name:    "command error",
			input:   "Command Error.",
			wantNil: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:    "partial response",
			input:   "#>11000101",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone := ParseZoneResponse(tt.input)
			if tt.wantNil {
				if zone != nil {
					t.Errorf("expected nil, got %+v", zone)
				}
				return
			}
			if zone == nil {
				t.Fatal("expected zone, got nil")
			}
			if zone.Zone != tt.wantZone {
				t.Errorf("zone = %q, want %q", zone.Zone, tt.wantZone)
			}
		})
	}
}

func TestParseZoneResponseAttributes(t *testing.T) {
	zone := ParseZoneResponse("#>1100010100150710100601")
	if zone == nil {
		t.Fatal("expected zone, got nil")
	}
	if zone.Zone != "11" {
		t.Errorf("Zone = %q, want 11", zone.Zone)
	}
	if zone.PA != "00" {
		t.Errorf("PA = %q, want 00", zone.PA)
	}
	if zone.Power != "01" {
		t.Errorf("Power = %q, want 01", zone.Power)
	}
	if zone.Mute != "01" {
		t.Errorf("Mute = %q, want 01", zone.Mute)
	}
	if zone.DT != "00" {
		t.Errorf("DT = %q, want 00", zone.DT)
	}
	if zone.Volume != "15" {
		t.Errorf("Volume = %q, want 15", zone.Volume)
	}
	if zone.Treble != "07" {
		t.Errorf("Treble = %q, want 07", zone.Treble)
	}
	if zone.Bass != "10" {
		t.Errorf("Bass = %q, want 10", zone.Bass)
	}
	if zone.Balance != "10" {
		t.Errorf("Balance = %q, want 10", zone.Balance)
	}
	if zone.Channel != "06" {
		t.Errorf("Channel = %q, want 06", zone.Channel)
	}
	if zone.Keypad != "01" {
		t.Errorf("Keypad = %q, want 01", zone.Keypad)
	}
}

func TestIsErrorResponse(t *testing.T) {
	if !IsErrorResponse("Command Error.") {
		t.Error("expected true for 'Command Error.'")
	}
	if !IsErrorResponse("Command Error.\r\n") {
		t.Error("expected true for 'Command Error.\\r\\n'")
	}
	if IsErrorResponse("#>1100010100150710100601") {
		t.Error("expected false for zone response")
	}
}

func TestQueryCommand(t *testing.T) {
	if cmd := QueryCommand(1); cmd != "?10\r" {
		t.Errorf("QueryCommand(1) = %q, want ?10\\r", cmd)
	}
	if cmd := QueryCommand(2); cmd != "?20\r" {
		t.Errorf("QueryCommand(2) = %q, want ?20\\r", cmd)
	}
}

func TestControlCommand(t *testing.T) {
	cmd := ControlCommand("11", "vo", "15")
	if cmd != "<11vo15\r" {
		t.Errorf("ControlCommand = %q, want <11vo15\\r", cmd)
	}
}

func TestBaudRateCommand(t *testing.T) {
	cmd := BaudRateCommand(115200)
	if cmd != "<115200\r" {
		t.Errorf("BaudRateCommand = %q, want <115200\\r", cmd)
	}
}

func TestResolveAttribute(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"power", "pr", true},
		{"pr", "pr", true},
		{"volume", "vo", true},
		{"mute", "mu", true},
		{"treble", "tr", true},
		{"bass", "bs", true},
		{"balance", "bl", true},
		{"channel", "ch", true},
		{"source", "ch", true},
		{"keypad", "ls", true},
		{"invalid", "", false},
	}
	for _, tt := range tests {
		got, ok := ResolveAttribute(tt.input)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ResolveAttribute(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestValidZoneID(t *testing.T) {
	tests := []struct {
		zone     string
		ampCount int
		valid    bool
	}{
		{"11", 1, true},
		{"16", 1, true},
		{"21", 1, false},
		{"21", 2, true},
		{"26", 2, true},
		{"31", 2, false},
		{"31", 3, true},
		{"10", 1, false},
		{"17", 1, false},
		{"00", 1, false},
		{"", 1, false},
		{"1", 1, false},
	}
	for _, tt := range tests {
		got := ValidZoneID(tt.zone, tt.ampCount)
		if got != tt.valid {
			t.Errorf("ValidZoneID(%q, %d) = %v, want %v", tt.zone, tt.ampCount, got, tt.valid)
		}
	}
}

func TestBaudRateSteps(t *testing.T) {
	steps := BaudRateSteps(9600, 115200)
	expected := []int{19200, 38400, 57600, 115200}
	if len(steps) != len(expected) {
		t.Fatalf("BaudRateSteps(9600, 115200) = %v, want %v", steps, expected)
	}
	for i, v := range steps {
		if v != expected[i] {
			t.Errorf("step[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestAmpIDForZone(t *testing.T) {
	if id := AmpIDForZone("11"); id != 1 {
		t.Errorf("AmpIDForZone(11) = %d, want 1", id)
	}
	if id := AmpIDForZone("26"); id != 2 {
		t.Errorf("AmpIDForZone(26) = %d, want 2", id)
	}
}
