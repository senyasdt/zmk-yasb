# YASB widget

This repo does not need a custom Python YASB widget. It uses YASB's built-in `yasb.custom.CustomWidget` and reads the JSON file written by `zmk-helperd`.

1. Add `zmk_layer_widget.yaml` under the `widgets:` section in your YASB config.
2. Add `zmk_layer_styles.css` to your YASB stylesheet.
3. Put `zmk_layer` into the relevant bar widget list, for example:

```yaml
bars:
  primary-bar:
    widgets:
      left:
        - zmk_layer
```

The helper must be running and writing:

```text
%APPDATA%\zmk-yasb\state.json
```

The widget expects this JSON shape:

```json
{
  "label": "SYMBOLS",
  "top": 1,
  "name": "SYMBOLS",
  "effective": "0, 1",
  "temp": "1",
  "default": "0",
  "connected": true,
  "status": "ON",
  "battery": 87,
  "battery_label": "87%",
  "battery_left": -1,
  "battery_right": 87,
  "battery_halves": "L ? / R 87%",
  "battery_left_label": "?",
  "battery_right_label": "87%",
  "tooltip": "Status: ON\nBattery: L ? / R 87%\nTop: SYMBOLS\nEffective: 0, 1\nTemporary: 1\nDefault: 0"
}
```
