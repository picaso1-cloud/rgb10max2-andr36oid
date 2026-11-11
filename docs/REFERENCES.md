# REFERENCES — curated sources & tools

**Date:** 2026-11-08 · Part of the master plan (`MASTER-PLAN.md`)

Purpose: keep every source we actually use in one place, say **what it is good
for**, which device it applies to, and whether we have verified it. Wiki claims
are **[inferred]** until observed on our hardware.

---

## 1. Reverse engineering / method

| Source | What it gives us | Status |
|---|---|---|
| [aicodeking/leanrea](https://github.com/aicodeking/leanrea) | 8 KB skill: fastest plain-CLI path for binaries/APKs/firmware, evidence tagging (`seen`/`inferred`/`unknown`), batching rules | ✅ adopted → `docs/RE-METHOD.md` |
| [morluto/rea](https://github.com/morluto/rea) | "Reverse Engineer Anything" MCP; reference for tool routing (we use it as a catalogue, not a dependency) | reference |
| `binwalk` / `unblob`, `jadx`, `apktool`, `dtc`, `fdtoverlay`, `mtools`, `readelf`, `objdump` | Our actual extraction/analysis toolchain | installed/used |

---

## 2. Linux distributions / kernels

| Source | What it gives us | Devices | Status |
|---|---|---|---|
| [ROCKNIX/distribution](https://github.com/ROCKNIX/distribution) | Shared Linux core: RK3326 + RK3566 images, quirks, build system, kernel/U-Boot, modern Mesa/glibc, sway, ARMSX2 packaging | RGB10 Max 2, RG353V | ✅ RG353V supported; Max 2 port in progress |
| [rocknix.org](https://rocknix.org) | Device wiki, install rules, quirks documentation | both | reference |
| [archr-linux/Arch-R](https://github.com/archr-linux/Arch-R) | Arch/pacman alternative core; RK3326 device definitions and build system | RGB10 Max 2 (RGB10, not Max 2) | plan B for pacman |
| [christianhaitian/arkos](https://github.com/christianhaitian/arkos) | ArkOS reference: boot chain, RG353V quirks, update/flash scripts | RG353V | reference/legacy |
| [Batocera.linux](https://batocera.org) | Max 2 support reference; simpler stack | RGB10 Max 2 | reference |
| [torvalds/linux](https://github.com/torvalds/linux) | Mainline DTS (`rk3326-odroid-go3`, `rk3566-anbernic-rg353v`), panel/joypad drivers | all | ✅ used |
| [linux-firmware](https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git) | Wi-Fi/BT firmware (`rtl8723bu_nic.bin`, etc.) | all | ✅ used |

---

## 3. Android / firmware

| Source | What it gives us | Devices | Status |
|---|---|---|---|
| [TheGammaSqueeze/GammaOS](https://github.com/TheGammaSqueeze/GammaOS) | Debloated LineageOS base, partition/flash procedure, T618 + RK3566 support | RG353V, RG405V | ✅ RG353V running GammaOS Lite 12 |
| [TheGammaSqueeze/GammaOSNext](https://github.com/TheGammaSqueeze/GammaOSNext) | Android 13/14 GSI for T618 (RG405M/V/RG505) | RG405V | target: v1.1 T618 (Android 14) |
| [GammaOS Next installation wiki](https://github.com/TheGammaSqueeze/GammaOSNext/wiki/GammaOS-Next-Installation) | Exact unlock/flash/erase/rollback steps per device | RG405V | ✅ used |
| [unisoc-unlock](https://pypi.org/project/unisoc-unlock/) / [subut-rehost](https://thegammasqueeze.github.io/subut-rehost/) | Unisoc bootloader unlock (Linux/macOS/Chrome) | RG405V | reference |
| [RG405V unbricker](https://drive.google.com/file/d/1idODOwE197mRHwMHDU_Y7bnAiWiackbN/view) | Stock restore after a bad flash (Windows `UpgradeDownload.exe`) | RG405V | **must verify before flashing** |
| [gammaos.org](https://gammaos.org) | Compatibility table, install/flash guidance | RG353V, RG405V | reference |
| [andr36oid](https://github.com/andr36oid) | Android 11 base used on the RGB10 Max 2 | RGB10 Max 2 | ✅ ported |
| [LineageOS GSI / Treble](https://source.android.com/docs/core/architecture/gsi) | GSI strategy for Android upgrades | RGB10 Max 2 (research), RG353V (experiment) | research |
| Magisk | Root/persistence (`/data/adb/service.d/`) | RG353V (installed), others | ✅ used |
| `adb` / `fastboot` | Recon, flashing, backups | all | ✅ used |
| `rkdeveloptool` / SDDiskTool / RKDevTool | Rockchip maskrom/loader/flash + recovery cards | RGB10 Max 2, RG353V | ✅ documented |
| Unisoc tools (`spd_dump`, Unisoc Flash Tool) | T618 firmware backup/flash | RG405V | [unknown] verify before use |

---

## 4. Device-specific references

| Source | What it gives us | Status |
|---|---|---|
| ROCKNIX RG353 quirk scripts (`Anbernic RG353V`) | GPIO/WiFi/LED/audio paths, panel v1/v2 detection | ✅ seen |
| ROCKNIX RK3326 `a_boot.ini` / `b_boot.ini` | ADC→DTB selection; Max 2 = OGS (ADC 85) | ✅ seen |
| Mainline `rk3326-odroid-go3.dts` + `rk3326-odroid-go.dtsi` | Panel (kd50t048a, 2 lanes), joypad mux, keys, charger | ✅ used |
| Mainline `rk3566-anbernic-rg353v.dts` + `rk3566-anbernic-rg353x.dtsi` | RG353V board support | ✅ used |
| GammaOS decompiled `boot.dts` / `dtbo` (our `rg353v-internals/survey/`) | Panel v1/v2, vendor quirks | ✅ seen |
| Anbernic product pages | Specs (T618, 4 GB, 128 GB, Android 12) | [inferred] verify on device |
| ROCKNIX `boot.scr` / extlinux | RK3326 boot flow, overlay mechanism | ✅ seen |

---

## 5. Emulation / apps

| Source | What it gives us | Status |
|---|---|---|
| [ARMSX2](https://github.com/ARMSX2/ARMSX2) | Native ARM64 PS2 (PCSX2 fork); ROCKNIX package + AppImage | ✅ RG353V (preinstalled) |
| AetherSX2 / ARMSX2 APK | PS2 on Android (RG353V/RG405V) | ✅ user uses it |
| [RetroArch](https://www.retroarch.com/) | Libretro front-end/cores | ✅ |
| [Dolphin](https://dolphin-emu.org/), [PPSSPP](https://www.ppsspp.org/), Yabasanshiro | GC/Wii, PSP, Saturn | reference |
| [Winlator](https://github.com/brunodev85/winlator) / box64 | x86 Windows games on ARM | research |
| [Moonlight](https://moonlight-stream.org/) + [Sunshine](https://github.com/LizardByte/Sunshine) | Remote play | ✅ RGB10 Max 2 |
| [Tailscale](https://tailscale.com/) | Mesh VPN / remote access | ✅ RGB10 Max 2; target others |
| EmulationStation / Daijishō / Beacon | Front-ends | ✅ ES (ROCKNIX/ArkOS), Daijishō (GammaOS) |

---

## 6. Monitoring / management (no root)

| Source | What it gives us | Devices | Status |
|---|---|---|---|
| **[Zaruber/ohmyanbernic](https://github.com/Zaruber/ohmyanbernic)** | Termux web dashboard: CPU/RAM/storage/battery, file manager (home only), process manager, Python script runner; zero dependencies (stdlib) | RG405V (designed for it), any Android + Termux | ✅ reference analysed |
| Termux | Linux userland on Android without root; SSH, Python, wake-lock | all Android devices | reference |
| `scrcpy` | Screen mirroring/control over ADB | all | reference |

### Notes on `ohmyanbernic` (analysed 2026-11-08)

- **Stack:** Python 3 stdlib only (`http.server`, `subprocess`, `threading`);
  `server.py` + one-page `index.html`; port 8080; SSH via Termux (8022).
- **Data sources:** `/proc/loadavg`, `/proc/meminfo`, `df -h`, `/proc/uptime`,
  `termux-battery-status` (fallback `/sys/class/power_supply/battery/{capacity,
  status,temp}`), `ps aux`, `hostname -I`/`ip route`. No root.
- **Useful for us:** unrooted RG405V recon and day-to-day telemetry; a model
  for a tiny "andr36oid status" service; script runner for experiments.
- **Caveats:** Android kills Termux in the background unless battery
  optimization is disabled and `termux-wake-lock` is set; file manager is
  sandboxed to `$HOME`; `kill` works only on the same UID without root;
  local-network only; the CPU metric is load average (not per-core usage).
- **Not adopted as a dependency** — it is a reference/optional tool, not part of
  the OS image.

---

## 7. Community / guides

| Source | Use |
|---|---|
| [Retro Game Corps guides](https://retrogamecorps.com/) | Setup/hardware background, card advice |
| [ROCKNIX Wiki](https://rocknix.org) | Device support/install/quirk docs |
| [ArkOS wiki](https://github.com/christianhaitian/arkos/wiki) | RG353-specific flashing/BIOS/emulator notes |
| [GammaOS docs](https://gammaos.org) | Device compatibility, flash/rollback |
| [Arch-R docs](https://arch-r.io) | Arch-R device list/build notes |

---

## 8. Rule for adding a reference

Add a row only if we have **used** it or plan to use it for a concrete task.
Include: link, what it gives us, devices, status (✅ used / reference /
[unknown]). If a source contradicts hardware, hardware wins; tag the claim.
