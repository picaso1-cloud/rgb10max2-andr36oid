# 01 — Hardware

## Board identity

| Item | Value |
|---|---|
| Product | Powkiddy RGB10 Max 2 (`hwrev=v10-go3` in stock firmware) |
| SoC | Rockchip rk3326 (Cortex-A35 ×4, Mali-G31 MP2) |
| RAM | 1 GB LPDDR (plus ~730 MB zram swap from Android's fstab) |
| Boot path | boots the `rk3326-odroidgo3` device-tree path (same family as RG351 devices) |
| Stock firmware | RetroOZ / Arteroid-class Linux (ground-truth logs captured in a working boot) |
| Panel | `elida,kd50t048a` 480×854 portrait, MIPI-DSI, **2 lanes**, 251-level PWM backlight |
| Orientation | panel is natively landscape-in-portrait; Android needs `androidboot.sf_orientation=ORIENTATION_270` on the kernel cmdline (added in `boot.ini`) |

## The two USB-C ports (important!)

| Port | Marking | Wiring | Role |
|---|---|---|---|
| 1 | **POWER** | 5 V + CC only, **no data lines** | charging only |
| 2 | **OTG** | full USB 2.0 data + 5 V on the shared VBUS net | host devices / the internal hub bus |

**They share one 5 V net** (the OTG VBUS / hub supply). This single fact explains
most of the board's power behavior — see [02-power-and-charging.md](02-power-and-charging.md).

Never treat the OTG port as a charge input: a charger event there triggers the
PMIC's OTG priority logic and kills the hub (Wi-Fi drops).

## Internal USB topology

- The **dwc2** controller (`/dev` `usb@ff300000`) is the only active USB controller in the
  ported configuration. Behind it: an internal **hub (Genesys 05e3:0608)** with
  the **Wi-Fi/BT combo dongle (Realtek 0bda:b720, RTL8723BU class)** and room
  for external devices.
- Enumeration evidence in a healthy boot: `usbdevs=8` (hub + dongle + ports).
- The dedicated host pair (`usb_host0_ehci/ohci`) can be enabled in the DTS but
  the board's external connector is wired to the dwc2 bus.

## Wi-Fi / Bluetooth

- Module: Realtek **RTL8723BU** (2.4 GHz Wi-Fi + BT 4.0 combo on USB).
- Driver: `rtl8xxxu` (mainline) — module must be preloaded or it binds late;
  see [05-wifi.md](05-wifi.md).
- Hardware Wi-Fi switch: a physical slider cuts the module's power/USB visibility.
  On = module enumerates; Off = disappears from the bus. The software must not
  assume the module is always present.

## Input

- Joypad: `odroidgo3-joypad` node, ADC-amux analog sticks, `invert-absrx` on
  the right stick X (see below for the final polarity scheme), explicit amux
  channels **replaced by `amux-channel-mapping = <1 0 2 3>`** (see below).
- Face buttons/Start/Select: sw9=315 (Start), sw10=314 (Select) verified.
- Stick pushes: sw11/sw12 emit **BTN_THUMBL (317) / BTN_THUMBR (318)** — real
  L3/R3 (needed for the Select+Start-adjacent mouse-mode chords and emulators).
- **Right-stick mux wiring differs from the R36S**: mux ch0 = right-X,
  ch1 = right-Y (R36S is the reverse). The odroidgo3 driver reads
  `amux-channel-mapping` (axis order RY,RX,Y,X) — set it to `<1 0 2 3>` and
  keep `invert-absrx`, no `invert-absry`. Without this the right stick is
  swapped and inverted in everything (mouse mode, emulators).
- The stock `abs_rx_channel`/`abs_ry_channel` DT properties are **inert** in
  this driver — they belong to RetroOZ's kernel and mislead if copied.
- No touchscreen (panel is non-touch).

## Mouse mode (com.gameconsole.joymouse)

- Native root daemon (`/system/bin/joyMouse`), started by init at boot; watches
  `/dev/input/event3` and **EVIOCGRABs the pad only while mouse mode is ON**
  (that's also why getevent sees nothing while it's active).
- Toggle: hold **Select+Start** ~1 s (the chord requires centered sticks —
  "toggle chord ignored: a stick is being pushed").
- Pointer = **right stick**, scroll = left stick; click mapping and per-button
  actions configurable in the JoyMouseSettings app.
- Config = `persist.sys.joymouse.*` properties, read live. Final state:
  `pointer_stick=right`, `invert_ry=false`, `pointer_invert_x=false` — all
  axis correction happens in the kernel DTS.
- The engine skips its own right-Y flip when `ro.product.device` == "r36s"
  (ours reports r36s); our polarity correction is therefore kernel-side.

## Battery / PMIC

- PMIC: **Rockchip rk817** at i2c address `0x20` on `i2c-0` (`ff180000.i2c`).
- Battery: 4000 mAh design (go3 values), fuel gauge is a separate driver from
  the charger — the battery % stays honest even when the charger driver is
  confused.
- Charge LED on `gpio0 PB5`; no vibrator on this board.
- Charger DC-detect pin `gpio0 PB3` (`dc_det_gpio`) — **must stay enabled**;
  it is what lets the charger enable the charge path (removing it disables
  charging entirely — verified in v21/v24 experiments).

## Storage

- SD-only boot (no eMMC on this unit). Card GPT: p1 BOOT (FAT, starts at
  sector **32768**), p6 `userdata` (f2fs), p8 `vendor` (ext4), `/` = p2 (erofs,
  read-only), plus swap via **zram** (no swap partition).
