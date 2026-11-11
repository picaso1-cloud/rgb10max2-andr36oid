# 11 — ROCKNIX on the RGB10 Max 2 (RK3326): port analysis

**Date:** 2026-11-08 · **Core decision:** ROCKNIX is the shared Linux core
(`docs/DECISIONS.md` ADR-001). This doc is the Max 2 slice.

Legend: **[seen]** verified here · **[inferred]** reasoned/public · **[unknown]**
to probe on hardware.

---

## 1. The key hardware fact

The RGB10 Max 2 identifies itself as an **ODROID-GO Super class board**:
RetroOZ boots it with `hwrev=v10-go3` / `rk3326-odroidgo3-linux.dtb`
("Machine model: Hardkernel ODROID-GO3"). [seen: `device-work/kernel/rk3326-rgb10max2-android.dts` header, from the stock card]

ROCKNIX's RK3326 `a_boot.ini` maps `hwrev=v10-go3` to ADC **85** and selects
`rk3326-odroid-go3.dtb`. [seen: `projects/ROCKNIX/devices/RK3326/packages/u-boot/config/a_boot.ini`]

**Consequence:** the standard **ROCKNIX-RK3326-a** image should boot the Max 2
using the ODROID-GO Super device tree — no custom bootloader or full rebuild
needed for first contact.

---

## 2. What the OGS device tree already gives us

| Component | OGS (ROCKNIX) | Max 2 | Match |
|---|---|---|---|
| Panel | `elida,kd50t048a` + `sitronix,st7701`, 2 DSI lanes, 480×854, rotation 270 | same panel family, 2 lanes, 480×854, rotation 270 | ✅ [seen] st7701 `kd50t048a_desc.lanes = 2` |
| Backlight | PWM (`pwm1`), 251 levels | same | ✅ [seen] |
| PMIC / charger | RK817 charger + simple-battery | RK817 | ✅ [inferred] |
| Sticks | `rocknix-singleadc-joypad`, amux GPIO3 PB3/PB0/PB5, SARADC ch1 | same mux pins | ✅ [seen] |
| D-pad / face buttons | GPIO1 PB4–PB7, PA2/PA5/PA6/PA7 | same pins | ✅ [seen] |
| Volume | GPIO2 PA0/PA1 | same | ✅ [seen] |
| USB / internal hub | `usb20_otg` host; `u2phy_otg` disabled; `vcc_host` 5 V always-on | internal Genesys hub + RTL8723BU Wi-Fi | ⚠️ hub power to verify |
| Audio | RK817 codec, SPK/HP | same | ✅ [inferred] |
| Wi-Fi | OGS has none | **RTL8723BU USB** (`0bda:b720`) | ⚠️ driver + power |

### Linux drivers present in ROCKNIX RK3326

- `CONFIG_RTL8XXXU=m` — upstream Realtek USB driver; its Kconfig lists
  **RTL8723BU as known-to-work**, so the Wi-Fi chip should bind. [seen: kernel
  config + mainline Kconfig]
- Firmware **present**: `rtlwifi/rtl8723bu_nic.bin` (the file `rtl8xxxu` loads;
  `rtl8723bu_bt.bin` is not needed for Wi-Fi). User firmware can also be dropped
  in `/storage/.config/firmware/`. [seen: image firmware tree + driver source]
- `CONFIG_RTW88_8723DU=m`, `CONFIG_RTL8723BS=m` — alternatives.
- `CONFIG_ROCKNIX_SINGLEADC_JOYPAD` (driver patch) supports per-axis tuning
  (`abs_rx-p-tuning`, …) but **not** `amux-channel-mapping` /
  `invert-absrx`. [seen: SM4450 driver patch]
  - This matters only if the right stick axes are swapped on the Max 2; the
    OGS wiring may already match (to test).

### Image facts (verified offline, RK3326-a 20261101)

| Item | Value |
|---|---|
| File | `ROCKNIX-RK3326.aarch64-20261101-a.img.gz` |
| SHA-256 | `4e96c2574e7856f9c35963cd8aba1c1ba40a87694b8676f42b8bb2e5efaa02ce` |
| Raw image | 2,198,863,872 B (2.05 GiB), **MSDOS** table |
| Partitions | p1 FAT32 `ROCKNIX` @ 16 MiB (2 GiB) · p2 ext4 `STORAGE` (32 MiB, grows) |
| Boot flow | legacy U-Boot → `boot.scr` (ADC→`fdtfile`) → `sysboot` → `extlinux/extlinux.conf` |
| Kernel | **7.1.2** (embedded initramfs) |
| Userland | glibc **2.44**, Mesa **26.2.4**, sway, EmulationStation |
| ARMSX2 | **not shipped** on RK3326 (expected: too slow for PS2) |
| DTBs | at the FAT root (not `device_trees/`); `rk3326-odroid-go3.dtb` present |
| Wi-Fi | `rtl8xxxu.ko` + `rtl8723bu_nic.bin` present |
| Battery (go3 DTB) | 4000 mAh, 2 A charge, 4.2 V, 180 mΩ, 3.5–4.1 V — verified matching the Max 2 cell |

`boot.scr` contents (verified by `strings`): `hwrev=v10-go3` → ADC **85** →
`fdtfile = rk3326-odroid-go3.dtb`. This is exactly what the Max 2 reports.

### The go3 DTB already has the Max 2 buttons — with wrong codes

The image's `rk3326-odroid-go3.dtb` contains a single
`/odroidgo3-joypad` node with `sw1..sw22` buttons. The pins match the Max 2
exactly (verified against the v31 DTS), but the codes are OGS-oriented:

| node | pin | OGS code | Max 2 meaning |
|---|---|---|---|
| sw5 | gpio1 PA2 | BTN_EAST 305 | A → BTN_SOUTH 304 |
| sw6 | gpio1 PA5 | BTN_SOUTH 304 | B → BTN_EAST 305 |
| sw11 | gpio2 PA2 | HAPPY3 706 | **Start** 315 |
| sw12 | gpio2 PA3 | BTN_THUMBL 317 | **Select** 314 |
| sw13 | gpio2 PA4 | BTN_THUMBR 318 | L3 317 |
| sw14 | gpio2 PA5 | HAPPY5 709 | R3 318 |
| sw15 | gpio2 PA6 | BTN_TL 310 | L2 312 |
| sw16 | gpio2 PA7 | BTN_TR 311 | R2 313 |
| sw19 | gpio3 PB1 | BTN_SELECT 314 | F1 704 |
| sw20 | gpio3 PB7 | BTN_TR2 313 | R1 311 |
| sw21 | gpio3 PB2 | BTN_TL2 312 | L1 310 |
| sw22 | gpio3 PB4 | BTN_START 315 | F2 705 |

**No rewiring or DTS replacement is needed** — a tiny DT overlay overrides the
`linux,code` properties at boot. Implemented and validated offline:

- `dts/rk3326-rgb10max2-keys-overlay.dts`
- `dts/rk3326-rgb10max2-keys.dtbo` (compiled)
- `dts/apply-rocknix-max2.sh /dev/sdX1` (backs up `extlinux.conf`, installs the
  overlay, appends it to `FDTOVERLAYS`)

Verified with `fdtoverlay` against the image's go3 DTB: sw5=304, sw6=305,
sw11=315, sw12=314, sw13=317, sw14=318, sw15=312, sw16=313, sw19=704,
sw20=311, sw21=310, sw22=705. [seen]

---

## 3. What is expected to work out of the box

With `rk3326-odroid-go3.dtb` on a ROCKNIX-RK3326-a card:

- ✅ Panel (same kd50t048a, driver forces 2 lanes), rotation, backlight.
- ✅ D-pad, A/B/X/Y, volume (same GPIO pins — but see A/B code swap below).
- ✅ Charging + battery (RK817 charger; same battery class).
- ✅ Audio (RK817 codec + OGS audio path quirk).
- ✅ Boot, storage (2-partition layout), ES front-end.
- ⚠️ Wi-Fi: driver present; **hub power** is the question (the Max 2's internal
  hub is fed by the RK817 OTG 5 V net — our Android port needed the raw i2c
  latch `0xB4=0x47` to power it; mainline `vcc_host` is always-on but may not
  feed the internal hub).
- ⚠️ A/B may be swapped: OGS maps A→`BTN_EAST`, B→`BTN_SOUTH`; the Max 2 DTS
  uses A→`BTN_SOUTH`, B→`BTN_EAST`. [seen: compare `rk3326-odroid-go.dtsi` vs
  our v31 DTS]

---

## 4. What needs a Max 2 device tree / quirk

The OGS tree **already contains every Max 2 button pin** (see the table in §2),
so the only thing needed for full controls is the code overlay. A dedicated
device tree/quirk is still wanted for **upstream** correctness: model name,
board detection (OGS and Max 2 share ADC 85), battery/OCV if they differ, the
charge LED, and Wi-Fi power handling.

| Key | GPIO | Code (Linux) |
|---|---|---|
| Start | `gpio2 PA2` | `BTN_START` |
| Select | `gpio2 PA3` | `BTN_SELECT` |
| L3 | `gpio2 PA4` | `BTN_THUMBL` |
| R3 | `gpio2 PA5` | `BTN_THUMBR` |
| L2 | `gpio2 PA6` | `BTN_TL2` |
| R2 | `gpio2 PA7` | `BTN_TR2` |
| F1 | `gpio3 PB1` | `BTN_TRIGGER_HAPPY1` (or `KEY_F1` for app use) |
| L1 | `gpio3 PB2` | `BTN_TL` |
| F2 | `gpio3 PB4` | `BTN_TRIGGER_HAPPY2` |
| R1 | `gpio3 PB7` | `BTN_TR` |
| D-pad | `gpio1 PB4–PB7` | `BTN_DPAD_*` (more standard than the Android `KEY_*`) |
| A/B/X/Y | `gpio1 PA2/PA5/PA6/PA7` | A=`BTN_SOUTH`, B=`BTN_EAST`, Y=`BTN_WEST`, X=`BTN_NORTH` |
| Volume | `gpio2 PA0/PA1` | `KEY_VOLUMEUP/DOWN` |

Also needed:

1. **Model/compatible**: `powkiddy,rgb10max2` + `rockchip,rk3326`.
2. **Battery**: 4000 mAh; copy the Max 2 OCV table from the v31 DTS if the OGS
   table differs.
3. **Charge LED**: `gpio0 PB5`, `linux,default-trigger = "battery-charging"`.
4. **Wi-Fi power** (if needed): a boot-time i2c poke to the RK817 OTG switch
   (replicate the Android latch: register `0xB4 = 0x47` on the RK817 i2c
   address `0x20`, bus varies) or a DT regulator/OTG fix.
5. **Right-stick axes**: verify; if swapped, patch the joypad driver to accept
   a channel map or remap at the evdev/ES level (axis tuning props exist).
6. **Board detection conflict**: OGS and Max 2 both report `hwrev=v10-go3` /
   ADC 85. A dedicated DTB cannot be auto-selected by ADC alone — options:
   - replace `rk3326-odroid-go3.dtb` on the card with the Max 2 variant (fine
     for our device; not upstream-safe);
   - add a boot.ini check on a Max 2-only GPIO (e.g. an extra key line) to pick
     the Max 2 DTB;
   - ship a ROCKNIX quirk that copies the Max 2 DTB over the go3 one when the
     device model is detected in userspace — too late for the current boot;
   - best upstream: `rk3326-powkiddy-rgb10max2.dtb` + a reliable board ID
     (GPIO strap / DMI-like ADC difference) — research needed.

---

## 5. Fast iteration method (no full image rebuild)

1. Flash **ROCKNIX-RK3326.aarch64-*-a.img** to an A2 SD.
2. On the PC, mount the FAT partition (first partition, label `ROCKNIX`).
3. **Apply the key overlay** with `dts/apply-rocknix-max2.sh /dev/sdX1` — it
   copies `rgb10max2-keys.dtbo` into `/overlays/` and adds it to the
   `FDTOVERLAYS` line (backup kept as `extlinux.conf.bak-rgb10max2`).
4. Boot, capture `dmesg`, `evtest`, `ip link`, battery/charger state.
5. If needed, iterate on the overlay only — no kernel/rootfs rebuild.

For a **dedicated DTB** (upstream work), compile
`rk3326-powkiddy-rgb10max2.dtb` from a Linux source tree with the ROCKNIX
RK3326 config, place it at the FAT root, and make `boot.scr`/`extlinux.conf`
select it (see §4 item 6 for the OGS/ADC collision). Keep the overlay method as
the quick path.

---

## 6. Test checklist (hardware)

- [ ] Boot logo → ES; panel orientation, resolution, backlight.
- [ ] All buttons + sticks in `evtest` / ES controller test:
      d-pad, A/B/X/Y, L1/L2/R1/R2, L3/R3, Start/Select, F1/F2, volume.
- [ ] Right stick direction (X/Y, no swap); deadzone.
- [ ] Wi-Fi: `ip link`, scan/connect; if the hub is dead, check `lsusb`
      (Genesys hub + 0bda:b720), then the RK817 OTG power/latch.
- [ ] Charging: plug the power port, `cat /sys/class/power_supply/*`,
      charge current; charge + Wi-Fi together (the Android lesson).
- [ ] Audio speaker + headphones; volume.
- [ ] Battery % and charge LED.
- [ ] Sleep/wake, suspend, reboot, cold boot.
- [ ] Emulation smoke test (PS1/N64/DC) + Moonlight/Tailscale if packaged.

---

## 7. Upstream path

1. Add `projects/ROCKNIX/devices/RK3326/linux/dts/rockchip/rk3326-powkiddy-rgb10max2.dts`
   based on `rk3326-odroid-go3.dts` + the table in §4.
2. Add `projects/ROCKNIX/packages/hardware/quirks/devices/Powkiddy RGB10 Max 2/`
   (device config, audio path, game configs).
3. Solve board detection (§4 item 6) so OGS users are not hijacked.
4. Submit a PR to ROCKNIX; keep the local DTB-override method as the fast path.

---

## 8. Open risks

- **Hub power** is the biggest unknown; the Android port needed a raw PMIC poke
  (`0xB4=0x47`) to feed the internal USB hub. Mainline `vcc_host` is always-on
  and `u2phy_otg` is already disabled (good signs), but the hub may still need
  the RK817 OTG rail. Test `lsusb` first.
- **Right-stick mapping** vs the ROCKNIX joypad driver (no channel-map property).
  The OGS wiring may already match; if not, patch the driver or remap in ES.
- **Board detection** collision with the ODROID-GO Super (both ADC 85).
- **Wi-Fi driver**: resolved offline — `rtl8xxxu.ko` and
  `rtl8723bu_nic.bin` are both present. [seen]
- **Charging topology** (two USB-C ports, shared 5 V net) may need the same
  stock-topology trick as Android — mainline already disables `u2phy_otg`,
  which is a good sign.
