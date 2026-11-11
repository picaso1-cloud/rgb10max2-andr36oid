# RGB10 Max 2 — unofficial andr36oid port

An **unofficial** port of [andr36oid](https://github.com/andr36oid) (Android 11 /
LineageOS 18.1, RK3326) to the **Powkiddy RGB10 Max 2** (`hwrev=v10-go3`), plus
the tools to build and flash it.

> **Not affiliated with andr36oid or Powkiddy.** This is a community port shared
> in the hope it is useful. No ROMs, BIOS files or proprietary vendor blobs are
> distributed here. Use at your own risk — flashing erases the card.

## Status: working, verified on real hardware

| Feature | State |
|---|---|
| Display / orientation (`kd50t048a`, `ORIENTATION_270`) | ✅ works |
| Buttons + both sticks (ODROID Go Super map), L3/R3 | ✅ works |
| Wi-Fi (RTL8723BU behind the internal hub) | ✅ works from battery, every cold boot |
| Bluetooth | ✅ works |
| Charging + honest battery % (no false charging) | ✅ works |
| OTG USB-C port (data, host devices, USB modes) | ✅ works |
| Power USB-C port | ⚠️ charge only (hardware limitation) |
| ADB over Wi-Fi | ✅ works |
| Root command daemon (live servicing) | ✅ works |

Known quirks are documented under [`docs/`](docs/), above all
[`docs/02-power-and-charging.md`](docs/02-power-and-charging.md): the two USB-C
ports share one 5 V net, and the Wi-Fi hub is powered from the rk817 OTG latch.

## Quick start

1. **Build the image** from an official andr36oid RK3326 image:

   ```sh
   ./scripts/build-rgb10max2-image.sh <andr36oid-rk3326.img.xz> -o RGB10MAX2-andr36oid.img
   ./scripts/build-rgb10max2-image.sh --download -o RGB10MAX2-andr36oid.img   # or fetch latest official
   ```

   The script patches only the `BOOT` and `vendor` partitions and verifies the
   result inside the image. It writes a `.sha256` next to the output.

2. **Flash the card** with our flasher (Linux/macOS; Windows experimental):

   ```sh
   cd flasher && ./build.sh          # or use the prebuilt binaries in dist/
   sudo ./dist/rgb10max2-flasher-linux-amd64 flash ../RGB10MAX2-andr36oid.img
   ```

   The flasher lists removable cards, asks for confirmation, writes the image and
   then **reads the card back and compares SHA-256** — cheap cards that corrupt
   data are caught immediately.

   Or use the official andr36oid SD flasher and point it at the image manually
   (its UI accepts a local image).

3. Insert the card, boot the console, pick **RGB10 Max 2** if the profile is not
   auto-detected.

Detailed steps and troubleshooting: [`docs/INSTALL.md`](docs/INSTALL.md).

## Repository layout

```
device/boot-profile/   BOOT profile files (DTBs, logo, u-boot/kernel dtbs)
dts/                   the device tree source + keys overlay + ROCKNIX apply script
service/               zz-diag.rc + diag.sh (Wi-Fi preload, PM pinning, root daemon)
tools/                 rk817tool.c, wifitool.c (+ prebuilt aarch64 binaries)
scripts/               build-rgb10max2-image.sh, migrate-console-state.sh
flasher/               cross-platform Go SD writer (Linux/macOS/Windows)
docs/                  hardware and port notes 01–11
```

## What the port changes

- **Device tree** derived from the r36s Android DTS: `kd50t048a` panel init and
  timings from the stock go3 tree, Go Super buttons, 251-level backlight,
  battery/charger values from go3, `dc_det_gpio` removed from the rk817 charger
  (fixes false charging), dwc2 kept in `dr_mode = "otg"`.
- **boot.ini**: `androidboot.sf_orientation=ORIENTATION_270` and
  `usbcore.autosuspend=-1`.
- **Wi-Fi power**: the RTL8723BU sits behind a hub powered from the rk817 OTG
  latch; the boot service flips it with `rk817tool` (raw `0xB4=0x47`), which is
  what makes Wi-Fi work from battery on every cold boot.
- **vendor**: `diag.sh` + `zz-diag.rc` preload `rtl8xxxu.ko`, pin USB runtime PM,
  and provide the root command daemon; `debug.sf.nobootanimation=1`.

## Privacy — what the image does and does not contain

The builder patches **only** two partitions of the official base image:

- `BOOT` (p1): profile files + `boot.ini` lines
- `vendor` (p8): `diag.sh`, `zz-diag.rc`, `rk817tool`, `wifitool`, one build.prop line

It never mounts or modifies `/data`. Therefore the produced image contains:

- ❌ no Tailscale authentication state (`/data/local/tmp/ts/state/…`)
- ❌ no Wi-Fi credentials or saved networks
- ❌ no apps, accounts, saves or any other per-install data
- ❌ no personal IP addresses, hostnames or keys (this repository was scrubbed before publication)

Tailscale is optional and installed by the user after first boot; its login state
stays on that console's `/data`. `scripts/migrate-console-state.sh` only runs when
you explicitly invoke it with your own state kit, and no state kit is published here.

## License and credits

- Port code, scripts and documentation: **GPL-3.0** (see [`LICENSE`](LICENSE)).
- Device trees are derived from Rockchip/mainline sources (GPL-2.0); the SPDX
  headers are preserved.
- The official andr36oid SD flasher is GPL-3.0 and is **not** copied here; we
  provide our own independent writer.
- andr36oid, LineageOS and their respective marks belong to their owners.
