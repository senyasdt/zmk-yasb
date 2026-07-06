package yasbstate

import "testing"

func TestFromReport(t *testing.T) {
	state := FromReport(1, 0x03, 0x01, 0x02, map[string]string{"1": "SYMBOLS"})
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
}
