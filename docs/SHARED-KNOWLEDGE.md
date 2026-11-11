# SHARED KNOWLEDGE — cross-device registry

**Date:** 2026-11-08 · Part of the master plan (`MASTER-PLAN.md`)

One place for every reusable finding, technique, command and trap. Per-device
detail stays in the device repos; this file is the **index + portability map**.

Evidence legend: **[seen]** observed here · **[inferred]** reasoned/public ·
**[unknown]** not probed. Always cite the source doc.

---

## 1. Platform inventory

| | RGB10 Max 2 | RG353V | RG405V |
|---|---|---|---|
| SoC | RK3326 (4×A35) | RK3566 (4×A55) | Unisoc T618 (2×A75+6×A55) |
| GPU | Mali-G31 | Mali-G52 | Mali-G52 MP2 |
| PMIC / charger | RK817 | RK817 | Unisoc PMIC (probe) [unknown] |
| Fuel gauge | separate driver (go3 values) [seen] | CW2015 (I2C 0x62) [seen] | probe [unknown] |
| Panel | elida kd50t048a, 480×854, 2-lane DSI [seen] | v2 elida kd35t133 640×480 (ID 0x3821) [seen] | 4″ 640×480 touch [inferred] |
| Input | adc-joystick + gpio keys, amux remap [seen] | retrogame_joypad + touch_joypad [seen] | probe [unknown] |
| Wi-Fi | SDIO, physical switch [seen] | Realtek SDIO [seen] | 2.4/5 GHz [inferred] |
| Storage | single microSD [seen] | eMMC + TF1 + TF2 [seen] | 128 GB eMMC + microSD [inferred] |
| OS | Android 11 (andr36oid) [seen] | GammaOS 12 (eMMC) + ArkOS (TF1) [seen] | Android 12 stock [inferred] |

---

## 2. Hardware / power / display

| ID | Finding | Applies to | Evidence | Source |
|---|---|---|---|---|
| H1 | rk817 OTG rail is a PMIC **latch**, set by raw i2c `0xB4=0x47` (`rk817tool`); survives warm reboot, resets on cold power-off | RGB10 Max 2 | [seen] | `rgb10max2-internals/02-power-and-charging.md` |
| H2 | Stock charging topology: `u2phy_otg` disabled, dwc2 phyless keeping `phys`, `dc_det` intact → charge + Wi-Fi simultaneously (+869 mA verified) | RGB10 Max 2 | [seen] | `rgb10max2-internals/02-power-and-charging.md` |
| H3 | With the latch armed, `dc_det` reads present → battery icon can say "charging"; % stays honest (separate gauge) | RGB10 Max 2 | [seen] | same |
| H4 | Black screen after boot = SF/HWC never enables CRTC; a sleep/wake cycle (KEYCODE_POWER ×2) after `sys.boot_completed` fixes it | RGB10 Max 2 | [seen] | `rgb10max2-internals/03-display.md` |
| H5 | Panel runs 480×854 portrait; Android uses ORIENTATION_270 | RGB10 Max 2 | [seen] | `01-hardware.md`, `03-display.md` |
| H6 | `debug.sf.nobootanimation=1` saves ~10 s boot | RGB10 Max 2 | [seen] | `06-android-internals.md` |
| H7 | Right stick on this board muxes differently: `amux-channel-mapping = <1 0 2 3>` + `invert-absrx`, **no** `invert-absry` | RGB10 Max 2 | [seen] | `04-input.md` |
| H8 | Stick clicks emit BTN_THUMBL/THUMBR (317/318) after the fix | RGB10 Max 2 | [seen] | `04-input.md` |
| H9 | Mouse mode = **Select+Start**; pointer = right stick; props `pointer_stick=right`, `invert_ry=false`, `pointer_invert_x=false` | RGB10 Max 2 | [seen] | `04-input.md` |
| H10 | RG353V panel variants: ID `0x3052` = v1 (`anbernic,rg353p-panel` + `newvision,nv3051d`), `0x3821` = v2 (`anbernic,rg353v-panel-v2`, vendor `elida,kd35t133`) | RG353V | [seen] | `rg353v-internals/09-kernel-dtb.md`, `14-rocknix-image-analysis.md` |
| H11 | RG353V boot model: TF1 OS card first, eMMC Android fallback, TF2 ROMs-only | RG353V | [seen] | `14-rocknix-image-analysis.md` §4 |
| H12 | RG353V quirks: WiFi GPIO 5, LED 77, PWM motor `pwmchip1`, jack 86, fake jacksense, audio card 1 / SPK-HP mux | RG353V | [seen] | ROCKNIX quirk scripts, `14-rocknix-image-analysis.md` §3 |
| H13 | RG405V: T618, 4 GB, 128 GB eMMC, 4″ 640×480, 5500 mAh, Android 12 | RG405V | [inferred] | anbernic.com product page |

---

## 3. Android layer (andr36oid)

| ID | Finding | Applies to | Evidence | Source |
|---|---|---|---|---|
| A1 | Wireless ADB: set `service.adb.tcp.port 5555` + restart adbd; persist via Magisk `/data/adb/service.d/` | RG353V | [seen] | `rg353v-internals/04-wireless-adb.md` |
| A2 | Root shell daemon: `/data/local/tmp/cmd.in` → `cmd.out` (no adb root needed) | RGB10 Max 2 | [seen] | `PROJECT-STATUS.md` quick ref |
| A3 | `gamemode`: raise CPU floor (RGB10 1.008 GHz; RG353V 816 MHz + DMC 780 MHz) | both | [seen] | `05-optimization.md`, `06-android-internals.md` |
| A4 | GPU `min_freq` writes are ignored while the GPU is suspended (devfreq); set via `max_freq` / policy instead | RG353V | [seen] | `rg353v-internals/05-optimization.md` |
| A5 | Janitor: periodic `am kill-all` (40 s RGB10 / 60 s RG353V) + force-stop heavy apps | both | [seen] | same |
| A6 | zram swap ~75% + swappiness 100 + readahead 512 + iostats off | RGB10 Max 2 | [seen] | `06-android-internals.md` |
| A7 | Package trim (191 packages) improves PMS scan and RAM | RGB10 Max 2 | [seen] | `06-android-internals.md` |
| A8 | Tailscale CLI (static aarch64) as daemon with socket; auth state migratable (`essentials.tgz`) | RGB10 Max 2 | [seen] | `PORT-REPORT.md`, `10-rebuild-guide.md` |
| A9 | Moonlight + Sunshine remote play stack | RGB10 Max 2 | [seen] | `PORT-REPORT.md` |
| A10 | GammaOS = debloated LineageOS/GSI + RK3566 vendor BSP; Magisk root; dynamic partitions | RG353V | [seen] | `02-gammaos-internals.md` |
| A11 | GammaOS Next v1.1 supports RG405M/V/RG505 (Android 13/14); RG353 stays on GammaOS 12 | RG405V / RG353V | [inferred] | gammaos.org, GammaOSNext repo |
| A12 | RG405V stock warning: third-party OS may not be flashable back to official → back up first | RG405V | [inferred] | anbernic.com notice |
| A13 | No-root Android telemetry (Termux): CPU/RAM/storage/battery, processes, script runner; battery via `termux-battery-status` or `/sys/class/power_supply/battery/*`; needs battery-optimization exemption + `termux-wake-lock` | all Android | [seen] | `REFERENCES.md` §6 (Zaruber/ohmyanbernic) |
| A14 | RG405V build: T618 (`ums512`), Android 12 **V1.23**, kernel **5.4.161**, A/B + dynamic partitions (77 parts), eMMC 116.5 GB, userdata 112.9 GB, super 5.6 GB | RG405V | [seen] | `rg405v-internals/02-recon-results.md` |
| A15 | RG405V locked/unrooted: `flash.locked=1`, verified boot green, `sys.oem_unlock_allowed=0`; `/system/xbin/su` exists but `setgid` fails; SELinux **Permissive** | RG405V | [seen] | same |
| A16 | RG405V panel is **480×640 portrait** 60 Hz; UI runs **640×480 landscape** (`rotation 1`, `user_rotation=1`); backlight `sprd_backlight` max 255 | RG405V | [seen] | same |
| A17 | RG405V input: `retrogame_joypad` (all gamepad keys + ABS X/Y/Z/RZ/GAS/BRAKE/HAT0), `touch_joypad`, `goodix-ts` (480×640), `gpio-keys` (vol/power), vibrator | RG405V | [seen] | same |
| A18 | RG405V perf: schedutil 614.4 MHz–**2.002 GHz** (3 policies), Mali-G52 **384–850 MHz** `simple_ondemand`, DDR scene 256–**1866 MHz**; zram 2.14 GB | RG405V | [seen] | same |
| A19 | RG405V Wi-Fi: Unisoc **SC2355 SDIO** (`sc2355_sdio_wlan`/`sprd_wlan`), BT combo (`sprdbt_tty`); 5 GHz 390/433 Mbps at probe | RG405V | [seen] | same |
| A20 | RG405V stock is a **Full/GApps** build (38 Google packages); ROM library (103 GB: PS2/PSP/PSX/Wii-GC/…) is on the **microSD**, internal `/sdcard` has emulator data (~4.8 GB) | RG405V | [seen] | same |
| A21 | **GammaOS Next RG405V** = v1.1 T618, **Android 14**, Lite (no GApps) ~2.3 GB split 7z; fresh install only, unlock via `unisoc-unlock`/subut-rehost, flash in fastbootd, mandatory userdata erase; rollback via RG405V unbricker | RG405V | [inferred] | GammaOS Next wiki/release |
| A22 | RG405V kernel **5.4.161** (clang 12, 2023-08-11); `/proc/config.gz` readable; **152** `=m` modules, 282 loaded; `KALLSYMS_ALL` (names visible, addresses hidden); **MODULE_SIG_FORCE** | RG405V | [seen] | `rg405v-internals/05-kernel-drivers-deep-dive.md` |
| A23 | RG405V power: **SC2730 PMIC** (SPI4) + **AW32257 charger** (I2C4 0x6a) + SC2730 FGU/Type-C/vibrator; charger-manager JEITA; battery **5000 mAh design, 4.2 V** | RG405V | [seen] | running DTS |
| A24 | RG405V display/input/WiFi/GPU: generic MIPI panel **480×640**; Goodix **GT9xx** touch; `singleadc-joypad` (`retrogame_joypad`); **SC2355 SDIO** WiFi (marlin3lite) + BT/GNSS/FM; **Mali-Gondul G52** 384–850 MHz | RG405V | [seen] | running DTS |
| A25 | RG405V security: bootloader locked, verity enforcing, SELinux **Permissive**; no root. **Shizuku-Next = shell UID 2000 (same as ADB)** — no extra kernel/debugfs access | RG405V | [seen] | live probe |
| A26 | RG405V clean boot timeline (USB ADB, Wi-Fi/BT off): SC2730 @0.47 s, mmc/eMMC/SD @0.8–0.96 s, Goodix GT9xx @0.91 s, joypad DEFER `-517`, Mali-Gondul @3.46 s, charger/AW32257/FGU/Type-C @3.68–3.76 s, SC2355 @3.94 s, DSI `480x640@60` @6.2 s. Wi-Fi debug flood is why early logs were lost before | RG405V | [seen] | `05-kernel-drivers-deep-dive.md` §8 |
| A27 | RG405V perf tuning (no root): Daijishō default home (`cmd package set-home-activity`), animations 0.5×, `deviceidle whitelist`, **AOT compile works via shell** (`cmd package compile -m speed -f`), safe reversible debloat | RG405V | [seen] | `06-optimization.md` |

**Porting the L3 layer:** the *logic* (sysctl values, floors, janitor loop,
service keep-alives) is portable; the *paths* (`/sys/devices/system/cpu/...`,
`/sys/class/devfreq/...`, package names, input device names) must be re-probed
per device. Never copy a path without verifying it exists.

---

## 4. Linux core

| ID | Finding | Applies to | Evidence | Source |
|---|---|---|---|---|
| L1 | ROCKNIX RK3566 Generic: glibc 2.44, Mesa 26.2.4, kernel 7.2.11, sway, ES | RG353V | [seen] | `14-rocknix-image-analysis.md` |
| L2 | ARMSX2 preinstalled (`/usr/share/armsx2-sa/armsx2-qt` + `start_armsx2.sh`), PS2 BIOS/savestate plumbing | RG353V | [seen] | same |
| L3 | ArkOS weston 5 lacks stable `xdg_wm_base` → Qt6 cannot create a window; fbdev is unmappable (`mmap fbmem: Invalid argument`) | RG353V ArkOS | [seen] | `12-ps2-linux-investigation.md` |
| L4 | ROCKNIX boot chain: idbloader `RKNS` @ 32 KiB, SPL DT @ 0x36EC0, U-Boot 2026.01 @ 8 MiB, FAT @ 16 MiB, `extlinux` with `FDTDIR` | RG353V | [seen] | `14-rocknix-image-analysis.md` §2 |
| L5 | U-Boot Anbernic Generic: SARADC model detect (RG353V = 695) + panel ID detect | RG353V | [seen] | same |
| L6 | ROCKNIX kernel 7.2.11 has embedded LZO initramfs; modules under `/usr/lib/kernel-overlays/base/lib/modules/7.2.11` (290 .ko) | RG353V | [seen] | same |
| L7 | ROCKNIX **does not list RGB10 Max 2**; port needs DTB + quirks (we have the v31 DTS) | RGB10 Max 2 | [inferred] | ROCKNIX device list |
| L8 | Arch-R supports RGB10 (not Max 2) and has **no RK3566 device** | both | [inferred] | Arch-R supported hardware |
| L9 | Batocera lists RGB10 Max/Max 2 support | RGB10 Max 2 | [inferred] | Batocera wiki |
| L10 | RG405V Linux: no mainline T618 handheld support → Android-only (default) | RG405V | [inferred] | community/upstream |
| L11 | RGB10 Max 2 identifies as **ODROID-GO Super class** (`hwrev=v10-go3`, ADC 85); ROCKNIX `a_boot.ini` selects `rk3326-odroid-go3.dtb` | RGB10 Max 2 | [seen] | `rgb10max2-internals/11-rocknix-rk3326-port.md` |
| L12 | OGS DTS already matches the Max 2 panel (`elida,kd50t048a` + `sitronix,st7701`, st7701 driver forces 2 lanes), RK817 charger, amux pins (GPIO3 PB3/PB0/PB5) and d-pad/face pins | RGB10 Max 2 | [seen] | same |
| L13 | ROCKNIX `singleadc-joypad` has per-axis tuning props (`abs_rx-p-tuning`, …) but **no** `amux-channel-mapping` / `invert-absrx` | RGB10 Max 2 | [seen] | driver patch |
| L14 | RTL8723BU (USB `0bda:b720`) is in `rtl8xxxu`'s known-to-work list; hub power comes from the RK817 OTG 5 V net (latch) | RGB10 Max 2 | [inferred] | kernel Kconfig + Android port |
| L15 | ROCKNIX RK3326 boot flow = legacy U-Boot + `boot.ini` (ADC→DTB) + `sysboot` extlinux; FAT DTB swap is the fast iteration path | RGB10 Max 2 | [seen] | ROCKNIX `a_boot.ini` |
| L16 | Max 2 extra buttons (Start/Select/L1/L2/R1/R2/L3/R3/F1/F2) are on GPIOs that the OGS tree maps to F3/F4/TL/TR/F1/F2 — pins match, codes differ | RGB10 Max 2 | [seen] | v31 DTS vs OGS dtsi |
| L17 | ROCKNIX RK3326-a image: kernel **7.1.2**, glibc 2.44, Mesa 26.2.4, no ARMSX2, `rtl8xxxu` + `rtl8723bu_nic.bin` present, `boot.scr` ADC 85 → `rk3326-odroid-go3.dtb` | RGB10 Max 2 | [seen] | `rgb10max2-internals/11-rocknix-rk3326-port.md` §2 |
| L18 | The image's go3 DTB already has all Max 2 button pins; only the `linux,code` values need an **overlay** (validated with `fdtoverlay`) | RGB10 Max 2 | [seen] | `rgb10max2-internals/dts/` |

**Shared Linux decision:** ROCKNIX is the only candidate that already covers
one device fully (RG353V) and shares the SoC family with the other (RK3326).
Port the Max 2 into it rather than maintaining two distros.

---

## 5. Build / release / verification

| ID | Technique | Evidence | Source |
|---|---|---|---|
| B1 | Verify FAT writes with **mtools** (`mdir -i img@@16777216 ::/`), not kernel vfat mounts (page-cache staleness + `--sizelimit` trap) | [seen] | `10-rebuild-guide.md` |
| B2 | Always `dd` + `sync` + checksum; verify the raw image, not just the `.gz` | [seen] | `10-rebuild-guide.md` |
| B3 | RGB10 Max 2 layout: p1 BOOT FAT, p2 system erofs, p6 userdata f2fs, p8 vendor ext4; zram swap | [seen] | `04-boot-chain.md` |
| B4 | ROCKNIX layout: p1 FAT (2 GiB) + p2 STORAGE ext4 (grows on first boot) | [seen] | `14-rocknix-image-analysis.md` |
| B5 | Reproducible builder: `image-patcher/build-rgb10max2-image.sh` (mtools BOOT rebuild); migration: `migrate-console-state.sh` | [seen] | `image-patcher/` |
| B6 | Prefer A2/quality cards; stock cards die and corrupt BOOT FAT | [seen] | `10-rebuild-guide.md` |
| B7 | GammaOS/ArkOS: do **not** use Etcher/Win32DiskImager; use the documented tools (SDDiskTool / dd / ApplePi-Baker) | [seen] | `rg353v-internals/07-rebuild-firmware.md` |
| B8 | Unisoc (RG405V): expected flash path is fastboot/Unisoc tool — must be verified before touching the device | [unknown] | RG405V recon |
| B9 | Max 2 controls fixed by a **DT overlay** (not a DTB replacement); `dts/apply-rocknix-max2.sh` installs it and edits `FDTOVERLAYS` with a backup | [seen] | `rgb10max2-internals/dts/` |

---

## 6. Reverse-engineering method (leanrea-inspired)

Adopted from `github.com/aicodeking/leanrea` (MIT). Full notes in
`docs/RE-METHOD.md`. Core rules:

1. **Work on a copy.** Extract into a scratch dir; never modify the source.
2. **Route by `file` / magic bytes first**, then cheapest inventory, then
   decompile only what is needed.
3. **Batch probes** into one shell call; cap output with `head`.
4. **Slice, don't `cat`** big/minified files.
5. **Mark every claim** seen / inferred / unknown; a missing fact is unknown,
   not absent.
6. **Static ≠ executed.** Reading code is [inferred] unless the behavior was
   observed.

Device-specific additions: dump partitions before flashing; hash every artifact;
keep a command log; record the exact model/firmware/SoC revision.

---

## 7. RE backlog (targets worth mining)

| Target | Why | Status |
|---|---|---|
| andr36oid Android 11 vendor BSP (RK3326) | init/HAL/services, perf knobs, boot.img, upgrade path | partial |
| GammaOS / GammaOS Next build + flasher | replicate debloat/tweaks, understand GSI/vendor split, rollback | partial |
| ROCKNIX build system + RK3326 quirks | shared Linux core, Max 2 port, ARMSX2 packaging | partial (RK3566) |
| Arch-R build system + device definitions | alternative core; RK3326 device template | studied |
| ArkOS boot chain + kernel patches | fallback/reference; legacy boot understanding | partial |
| RG405V stock/GammaOS Next (T618) | bootloader, partitions, Treble, sysfs, thermal, input | not started |
| Emulator stacks: ARMSX2/PCSX2, Dolphin, PPSSPP, Yabasanshiro, Winlator/box64 | compatibility/perf matrix per device | partial |
| Frontends: ES, Daijisho, Beacon | launcher choice, low-RAM behavior | partial |
| Streaming: Moonlight/Sunshine, Tailscale/WireGuard | best remote-play setup per device | partial |
| Browser/WASM gaming | what is actually playable | partial (`10-emulation-and-webasm.md`) |

---

## 8. Traps (learned the hard way)

| ID | Trap | Lesson |
|---|---|---|
| T1 | Hex codes misread once (0x03=ABS_RX, 0x04=ABS_RY) | verify every code against the header |
| T2 | Loop-mounted FAT appeared corrupt due to page cache / stale `--sizelimit` | use mtools for verification |
| T3 | `export LD_LIBRARY_PATH` poisons system binaries | launch with the loader `--library-path` only |
| T4 | GPU `min_freq` write "succeeds" but is ignored while suspended | check the actual policy |
| T5 | Wiki claim treated as fact before hardware verification (ROCKNIX/eMMC) | tag [inferred]; verify on device |
| T6 | Assumed multi-slot where the device has one slot | confirm hardware topology first |
| T7 | Old distro stack (weston 5 / fbdev) silently blocks Qt6 apps | check compositor protocols before debugging the app |
| T8 | Battery icon "charging" while latched is cosmetic, not a charger bug | check the gauge separately |
| T9 | Flashing a vendor Android image over a modified bootloader can be irreversible | back up + verify rollback before flashing |

---

## 9. Command shelf (copy-paste)

```sh
# --- Android (RGB10 Max 2 / GammaOS) ---
adb connect <ip>:5555
adb shell 'cat > /data/local/tmp/cmd.in << "EOF"\n{ id; getprop ro.build.version.release; } > /data/local/tmp/cmd.out 2>&1\nchmod 666 /data/local/tmp/cmd.out\nEOF'

# performance probes
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_available_governors
cat /sys/class/devfreq/*/available_frequencies
cat /sys/block/zram0/disksize
cat /proc/meminfo | grep -E 'MemTotal|MemAvailable|SwapTotal'

# --- Linux (ROCKNIX / ArkOS) ---
sha256sum -c image.sha256
gzip -dk image.img.gz
sudo dd if=image.img of=/dev/sdX bs=4M conv=fsync status=progress && sync
# verify FAT without mounting:
mdir -i /dev/sdX@@16777216 ::/

# --- RE quick starts (leanrea) ---
file -b target
readelf -dW elf | head
jadx -d out app.apk
unzip -l image.zip | head
strings -a -n 6 bin | head -400
rg -uu -o '.{0,200}NEEDLE.{0,200}' extracted/ | head
```

---

## 10. Maintenance

- Add a row whenever a finding is reusable on a second device.
- If a finding is device-specific, it still belongs here with its applicability.
- When a claim becomes hardware-verified, change **[inferred] → [seen]** and add
  the source.
- Review this file at the end of every session that touches hardware or a build.
