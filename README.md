# zmk-yasb

Windows bridge for showing a ZMK keyboard layer in YASB.

The YASB side keeps the same JSON contract as the existing `vial_layer` custom widget. The firmware sends numeric layer state over raw HID, and `zmk-helperd` writes `%APPDATA%\vial-helper\state.json` atomically.

## GitHub check

Before starting this repository I searched GitHub for existing `ZMK + YASB + layer status HID` implementations.

Closest matches found:

- `RaphCoder13/zmk-kblayerhelper`: ZMK module that sends layer information over raw HID.
- `zzeneg/zmk-raw-hid`: raw HID transport module for ZMK; this repo uses its event API.
- `hrmt-lab/zmk-rawhid-app`: broader raw HID app protocol for ZMK.

I did not find a ready-made YASB Windows helper using the same JSON contract. This repo keeps the host side in Go and includes a small ZMK module that follows the raw HID approach.

## Layout

- `cmd/zmk-helperd`: Windows helper daemon written in Go.
- `internal/hid`: minimal Windows HID enumeration and read code, no external Go dependencies.
- `internal/yasbstate`: YASB JSON state formatting and atomic writes.
- `modules/zmk-yasb-layer-status`: ZMK module that emits layer reports over raw HID.
- `yasb`: ready-to-paste YASB widget config and CSS.

## Report formats

The helper accepts two formats:

1. Native `zmk-yasb` binary report:

```text
byte 0: report id 0x7A
byte 1: top layer
byte 2-3: effective layer mask, little endian
byte 4-5: default layer mask, little endian
byte 6-7: temporary layer mask, little endian
```

2. Compatibility report from `zmk-kblayerhelper`:

```text
KBHLayer1\n
```

## ZMK firmware setup

This module depends on `zzeneg/zmk-raw-hid`, which provides:

```c
#include <raw_hid/events.h>
raise_raw_hid_send_event(...)
```

Add both the raw HID module and this module to `config/west.yml`:

```yaml
manifest:
  remotes:
    - name: zmkfirmware
      url-base: https://github.com/zmkfirmware
    - name: senyasdt
      url-base: https://github.com/senyasdt
    - name: zzeneg
      url-base: https://github.com/zzeneg
  projects:
    - name: zmk
      remote: zmkfirmware
      revision: main
      import: app/west.yml
    - name: zmk-yasb
      remote: senyasdt
      revision: main
    - name: zmk-raw-hid
      remote: zzeneg
      revision: main
  self:
    path: config
```

Enable it only on the USB-connected central half:

```conf
CONFIG_RAW_HID=y
CONFIG_RAW_HID_USAGE_PAGE=0xFF60
CONFIG_RAW_HID_USAGE=0x61
CONFIG_RAW_HID_REPORT_SIZE=32
CONFIG_ZMK_YASB_LAYER_STATUS=y
```

For split keyboards, the module CMake only builds when `CONFIG_ZMK_SPLIT_ROLE_CENTRAL` is true, matching the “central side only” constraint.

## Windows helper setup

Edit `config.example.json` for your keyboard VID/PID and layer names. Decimal values are accepted in JSON. For reference:

```json
{
  "vid": 7504,
  "pid": 24866,
  "usage_page": 65376,
  "usage": 97
}
```

Build on Windows:

```powershell
go build -o zmk-helperd.exe .\cmd\zmk-helperd
```

Run:

```powershell
.\zmk-helperd.exe -config .\config.json
```

You can also pass VID/PID as hex flags:

```powershell
.\zmk-helperd.exe -vid 0x1D50 -pid 0x6122
```

## YASB widget

Use the ready-to-paste files in `yasb/`, or add the same frontend contract manually:

```yaml
vial_layer:
  type: "yasb.custom.CustomWidget"
  options:
    label: "<span>⌨</span> {data[label]}"
    label_alt: "<span>⌨</span> {data[label]} · {data[effective]}"
    exec_options:
      run_cmd: 'cmd /c type %APPDATA%\vial-helper\state.json'
      run_interval: 250
      return_format: "json"
```

## Current limitations

- USB HID only.
- The firmware reports top layer, effective active-layer mask, default layer mask, and a derived temporary mask.
- The temporary mask is computed as `effective & ~default`, so it is a practical approximation rather than a separate ZMK source of truth.
- The Go helper is intended to be built and run on Windows. On other OSes it compiles only as a stub for the HID layer.
