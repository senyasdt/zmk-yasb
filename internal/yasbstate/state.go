package yasbstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type State struct {
	Label             string `json:"label"`
	Top               int    `json:"top"`
	Name              string `json:"name"`
	Effective         string `json:"effective"`
	Temp              string `json:"temp"`
	Default           string `json:"default"`
	Connected         bool   `json:"connected"`
	Status            string `json:"status"`
	Battery           int    `json:"battery"`
	BatteryLabel      string `json:"battery_label"`
	BatteryLeft       int    `json:"battery_left"`
	BatteryRight      int    `json:"battery_right"`
	BatteryHalves     string `json:"battery_halves"`
	BatteryLeftLabel  string `json:"battery_left_label"`
	BatteryRightLabel string `json:"battery_right_label"`
	Tooltip           string `json:"tooltip"`
}

func Offline() State {
	return State{
		Label:             "OFF",
		Top:               -1,
		Name:              "Keyboard disconnected",
		Effective:         "-",
		Temp:              "-",
		Default:           "-",
		Connected:         false,
		Status:            "OFF",
		Battery:           -1,
		BatteryLabel:      "-",
		BatteryLeft:       -1,
		BatteryRight:      -1,
		BatteryHalves:     "L ? / R ?",
		BatteryLeftLabel:  "?",
		BatteryRightLabel: "?",
		Tooltip:           "ZMK keyboard is not connected",
	}
}

func ConnectedUnknown() State {
	return State{
		Label:             "WAIT",
		Top:               -1,
		Name:              "Keyboard connected",
		Effective:         "-",
		Temp:              "-",
		Default:           "-",
		Connected:         true,
		Status:            "ON",
		Battery:           -1,
		BatteryLabel:      "-",
		BatteryLeft:       -1,
		BatteryRight:      -1,
		BatteryHalves:     "L ? / R ?",
		BatteryLeftLabel:  "?",
		BatteryRightLabel: "?",
		Tooltip:           "ZMK keyboard is connected; waiting for layer report",
	}
}

func FromReport(top int, effectiveMask, defaultMask, tempMask uint32, batteryRight, batteryLeft int, layers map[string]string) State {
	name := layerName(top, layers)
	effective := maskString(effectiveMask)
	temp := maskString(tempMask)
	def := maskString(defaultMask)
	batteryLabel := batteryString(batteryRight)
	batteryLeftLabel := batteryStringUnknown(batteryLeft)
	batteryRightLabel := batteryStringUnknown(batteryRight)
	batteryHalves := fmt.Sprintf("L %s / R %s", batteryLeftLabel, batteryRightLabel)
	return State{
		Label:             name,
		Top:               top,
		Name:              name,
		Effective:         effective,
		Temp:              temp,
		Default:           def,
		Connected:         true,
		Status:            "ON",
		Battery:           batteryRight,
		BatteryLabel:      batteryLabel,
		BatteryLeft:       batteryLeft,
		BatteryRight:      batteryRight,
		BatteryHalves:     batteryHalves,
		BatteryLeftLabel:  batteryLeftLabel,
		BatteryRightLabel: batteryRightLabel,
		Tooltip: fmt.Sprintf(
			"Status: ON\nBattery: %s\nTop: %s\nEffective: %s\nTemporary: %s\nDefault: %s",
			batteryHalves,
			name,
			effective,
			temp,
			def,
		),
	}
}

func batteryString(battery int) string {
	if battery < 0 || battery > 100 {
		return "-"
	}
	return fmt.Sprintf("%d%%", battery)
}

func batteryStringUnknown(battery int) string {
	if battery < 0 || battery > 100 {
		return "?"
	}
	return fmt.Sprintf("%d%%", battery)
}

func WriteAtomic(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func layerName(layer int, layers map[string]string) string {
	if name := layers[strconv.Itoa(layer)]; name != "" {
		return name
	}
	return fmt.Sprintf("LAYER %d", layer)
}

func maskString(mask uint32) string {
	if mask == 0 {
		return "-"
	}
	var layers []int
	for i := 0; i < 32; i++ {
		if mask&(1<<uint(i)) != 0 {
			layers = append(layers, i)
		}
	}
	sort.Ints(layers)
	parts := make([]string, 0, len(layers))
	for _, layer := range layers {
		parts = append(parts, strconv.Itoa(layer))
	}
	return strings.Join(parts, ", ")
}
