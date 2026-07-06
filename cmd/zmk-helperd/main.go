package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/senyasdt/zmk-yasb/internal/hid"
	"github.com/senyasdt/zmk-yasb/internal/yasbstate"
)

const (
	defaultUsagePage = 0xFF60
	defaultUsage     = 0x61
	reportID         = 0x7A
)

type config struct {
	VID         uint16            `json:"vid"`
	PID         uint16            `json:"pid"`
	UsagePage   uint16            `json:"usage_page"`
	Usage       uint16            `json:"usage"`
	Output      string            `json:"output"`
	Poll        string            `json:"poll"`
	Layers      map[string]string `json:"layers"`
	ReportBytes int               `json:"report_bytes"`
}

func defaultConfig() config {
	return config{
		UsagePage:   defaultUsagePage,
		Usage:       defaultUsage,
		Output:      filepath.Join(os.Getenv("APPDATA"), "vial-helper", "state.json"),
		Poll:        "250ms",
		ReportBytes: 32,
		Layers: map[string]string{
			"0":  "QWERTY",
			"1":  "SYMBOLS",
			"2":  "NUMBERS",
			"3":  "FUNCTIONS",
			"4":  "MANAGER",
			"5":  "NAV",
			"6":  "STACK",
			"7":  "UTIL",
			"8":  "MOUSE",
			"9":  "MOUSE-",
			"10": "MOUSE+",
			"11": "MOUSE-SCROLL",
		},
	}
}

func main() {
	var (
		configPath = flag.String("config", "", "path to config JSON")
		vidFlag    = flag.String("vid", "", "USB VID in hex, for example 0x1D50")
		pidFlag    = flag.String("pid", "", "USB PID in hex")
		once       = flag.Bool("once", false, "read one report and exit")
	)
	flag.Parse()

	cfg := defaultConfig()
	if *configPath != "" {
		if err := loadConfig(*configPath, &cfg); err != nil {
			log.Fatalf("load config: %v", err)
		}
	}
	if *vidFlag != "" {
		v, err := parseHex16(*vidFlag)
		if err != nil {
			log.Fatalf("parse -vid: %v", err)
		}
		cfg.VID = v
	}
	if *pidFlag != "" {
		v, err := parseHex16(*pidFlag)
		if err != nil {
			log.Fatalf("parse -pid: %v", err)
		}
		cfg.PID = v
	}
	cfg.Output = expandPercentEnv(cfg.Output)
	if cfg.VID == 0 || cfg.PID == 0 {
		log.Fatal("VID and PID are required; set them in config JSON or pass -vid/-pid")
	}

	poll, err := time.ParseDuration(cfg.Poll)
	if err != nil {
		log.Fatalf("parse poll duration: %v", err)
	}
	if poll < 50*time.Millisecond {
		poll = 50 * time.Millisecond
	}
	if cfg.ReportBytes < 8 {
		cfg.ReportBytes = 32
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := yasbstate.WriteAtomic(cfg.Output, yasbstate.Offline()); err != nil {
		log.Printf("write offline state: %v", err)
	}

	for {
		if err := run(ctx, cfg, poll, *once); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("helper loop: %v", err)
		}
		if *once || ctx.Err() != nil {
			return
		}
		if err := yasbstate.WriteAtomic(cfg.Output, yasbstate.Offline()); err != nil {
			log.Printf("write offline state: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(poll):
		}
	}
}

func run(ctx context.Context, cfg config, poll time.Duration, once bool) error {
	device, err := hid.OpenFirst(hid.Filter{
		VID:       cfg.VID,
		PID:       cfg.PID,
		UsagePage: cfg.UsagePage,
		Usage:     cfg.Usage,
	})
	if err != nil {
		return err
	}
	defer device.Close()
	log.Printf("connected: %s", device.Path())

	buf := make([]byte, cfg.ReportBytes)
	for {
		n, err := device.Read(buf)
		if err != nil {
			return err
		}
		report, ok := parseReport(buf[:n])
		if ok {
			state := yasbstate.FromReport(report.TopLayer, report.EffectiveMask, report.DefaultMask, report.TempMask, cfg.Layers)
			if err := yasbstate.WriteAtomic(cfg.Output, state); err != nil {
				return fmt.Errorf("write YASB state: %w", err)
			}
			if once {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
		}
		if n == 0 {
			time.Sleep(poll)
		}
	}
}

type layerReport struct {
	TopLayer      int
	EffectiveMask uint32
	DefaultMask   uint32
	TempMask      uint32
}

func parseReport(data []byte) (layerReport, bool) {
	if len(data) == 0 {
		return layerReport{}, false
	}

	textData := trimZeros(data)
	if strings.HasPrefix(string(textData), "KBHLayer") {
		line := strings.TrimSpace(string(textData))
		n, err := strconv.Atoi(strings.TrimPrefix(line, "KBHLayer"))
		if err != nil {
			return layerReport{}, false
		}
		return layerReport{TopLayer: n, EffectiveMask: layerBit(n)}, true
	}

	offset := 0
	if len(data) > 1 && data[0] == 0 && data[1] == reportID {
		offset = 1
	}
	if len(data[offset:]) < 6 || data[offset] != reportID {
		return layerReport{}, false
	}

	top := int(data[offset+1])
	effective := uint32(data[offset+2]) | uint32(data[offset+3])<<8
	defaultMask := uint32(0)
	tempMask := uint32(0)
	if len(data[offset:]) >= 8 {
		defaultMask = uint32(data[offset+4]) | uint32(data[offset+5])<<8
		tempMask = uint32(data[offset+6]) | uint32(data[offset+7])<<8
	}
	return layerReport{TopLayer: top, EffectiveMask: effective, DefaultMask: defaultMask, TempMask: tempMask}, true
}

func trimZeros(data []byte) []byte {
	end := len(data)
	for end > 0 && data[end-1] == 0 {
		end--
	}
	return data[:end]
}

func layerBit(layer int) uint32 {
	if layer < 0 || layer >= 32 {
		return 0
	}
	return 1 << uint(layer)
}

func loadConfig(path string, cfg *config) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, cfg)
}

func parseHex16(s string) (uint16, error) {
	s = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), "0x")
	v, err := strconv.ParseUint(s, 16, 16)
	return uint16(v), err
}

func expandPercentEnv(path string) string {
	for {
		start := strings.Index(path, "%")
		if start < 0 {
			return os.ExpandEnv(path)
		}
		end := strings.Index(path[start+1:], "%")
		if end < 0 {
			return os.ExpandEnv(path)
		}
		end += start + 1
		name := path[start+1 : end]
		value := os.Getenv(name)
		path = path[:start] + value + path[end+1:]
	}
}
