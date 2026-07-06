package yasbstate

import "testing"

func TestFromReport(t *testing.T) {
	state := FromReport(1, 0x03, 0x01, 0x02, 87, map[string]string{"1": "SYMBOLS"})
	if state.Label != "SYMBOLS" {
		t.Fatalf("label = %q, want SYMBOLS", state.Label)
	}
	if state.Effective != "0, 1" {
		t.Fatalf("effective = %q, want 0, 1", state.Effective)
	}
	if state.Default != "0" {
		t.Fatalf("default = %q, want 0", state.Default)
	}
	if state.Temp != "1" {
		t.Fatalf("temp = %q, want 1", state.Temp)
	}
	if !state.Connected {
		t.Fatal("connected = false, want true")
	}
	if state.BatteryLabel != "87%" {
		t.Fatalf("battery label = %q, want 87%%", state.BatteryLabel)
	}
}

func TestConnectedUnknown(t *testing.T) {
	state := ConnectedUnknown()
	if !state.Connected {
		t.Fatal("connected = false, want true")
	}
	if state.Status != "ON" {
		t.Fatalf("status = %q, want ON", state.Status)
	}
	if state.Battery != -1 {
		t.Fatalf("battery = %d, want -1", state.Battery)
	}
}
