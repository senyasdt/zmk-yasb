package main

import "testing"

func TestParseBinaryReport(t *testing.T) {
	report, ok := parseReport([]byte{0x7A, 3, 0x09, 0x00, 0x01, 0x00, 0x08, 0x00, 87, batteryMarker, 64})
	if !ok {
		t.Fatal("expected report to parse")
	}
	if report.TopLayer != 3 {
		t.Fatalf("top layer = %d, want 3", report.TopLayer)
	}
	if report.EffectiveMask != 0x09 {
		t.Fatalf("effective mask = 0x%X, want 0x09", report.EffectiveMask)
	}
	if report.DefaultMask != 0x01 {
		t.Fatalf("default mask = 0x%X, want 0x01", report.DefaultMask)
	}
	if report.TempMask != 0x08 {
		t.Fatalf("temp mask = 0x%X, want 0x08", report.TempMask)
	}
	if report.BatteryRight != 87 {
		t.Fatalf("right battery = %d, want 87", report.BatteryRight)
	}
	if report.BatteryLeft != 64 {
		t.Fatalf("left battery = %d, want 64", report.BatteryLeft)
	}
}

func TestParseCompatibilityReport(t *testing.T) {
	report, ok := parseReport([]byte("KBHLayer5\n\x00\x00"))
	if !ok {
		t.Fatal("expected report to parse")
	}
	if report.TopLayer != 5 {
		t.Fatalf("top layer = %d, want 5", report.TopLayer)
	}
	if report.EffectiveMask != 0x20 {
		t.Fatalf("effective mask = 0x%X, want 0x20", report.EffectiveMask)
	}
}
