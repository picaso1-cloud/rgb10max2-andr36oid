# 04 — Boot chain and card layout

## Card layout (GPT, SD only)

| Partition | Contents | FS | Notes |
|---|---|---|---|
| p1 | **BOOT** | FAT | starts at sector **32768** (verify with `cat /sys/block/mmcblk0/mmcblk0p1/start`) |
| p2 | `/` (system) | **erofs** | read-only filesystem — cannot remount rw; changing system needs an image rebuild |
| p6 | `/data` (userdata) | f2fs | mounted `rw,lazytime,noatime`; captures live in `/data/local/tmp` |
| p8 | `/vendor` | ext4 | **writable live** (`mount -o remount,rw /vendor`) — the injection point |
| — | swap | zram | `zramsize=75%` (~730 MB), no swap partition; `mkswap` on zram0 at boot is normal |

## Boot flow

1. SoC ROM → u-boot (`rg351mp-uboot.dtb` family names are the convention).
2. u-boot shows `logo.bmp` from BOOT.
3. `boot.ini` (BOOT partition, sourceable script):
   - selects the kernel + DTB (`rk3326-r36s-android.dtb` at top of BOOT),
   - passes `androidboot.sf_orientation=ORIENTATION_270`,
   - passes `usbcore.autosuspend=-1`.
4. Kernel boot: ~**2.7 s** to userspace init (fast; no low-hanging fruit).
5. init parses system → vendor rc files; `post-fs-data` starts the vendor
   service (our latch/panel/adb logic).
6. `on boot` → core/main classes → zygote (dual zygote 32+64 kept, no GApps).
7. `boot_completed` at ~48 s; launcher usable ~51 s; adb TCP up shortly after.

## Device-tree selection

The boot path lands on `rk3326-odroidgo3` names. The production DTS
([`dts/rk3326-rgb10max2-android.dts`](../dts/rk3326-rgb10max2-android.dts))
includes the kernel tree's `rk3326-r36s-android.dtsi` and changes only:

- panel → `elida,kd50t048a` (init sequence + 2 lanes + 251-level backlight),
- `invert-absrx` + explicit amux channels on the joypad,
- LEDs: no vibrator, charge LED gpio0 PB5,
- battery design capacity 4000, charger values from go3,
- `u2phy_otg` **disabled** + dwc2 `dr_mode="host"` keeping `phys=<&u2phy_otg>`
  (the stock-topology trick — see [02-power-and-charging.md](02-power-and-charging.md)),
- dedicated host pair (ehci+ohci) enabled,
- `dc_det_gpio` untouched.

### Compile flow (no kernel build needed)

```sh
git clone --depth 1 --filter=blob:none --sparse \
    https://github.com/andr36oid/android_kernel_gameconsole_r36s.git kernel-src
cd kernel-src && git sparse-checkout set arch/arm64/boot/dts/rockchip include

cpp -nostdinc -I include -I arch/arm64/boot/dts/rockchip -undef -D__DTS__ \
    -x assembler-with-cpp rk3326-rgb10max2-android.dts -o out.pp.dts
dtc -@ -I dts -O dtb -o out.dtb out.pp.dts
```

Then drop the `.dtb` at the top of BOOT as `rk3326-r36s-android.dtb` (and in
`Devices/RGB10MAX2/`). Verify with `fdtget`, e.g.:

```
fdtget out.dtb /syscon@ff2c0000/usb2-phy@100/otg-port status      # → disabled
fdtget out.dtb /usb@ff300000 dr_mode                              # → host
fdtget out.dtb /i2c@ff180000/pmic@20/charger dc_det_gpio -t x     # → 51 b 0
```

## Boot timing analysis (measured, Android 11, 1 GB)

| Phase | Time (from power-on) | Notes |
|---|---|---|
| kernel → init | 2.7 s | nothing to gain without a kernel rebuild |
| zygote service start | 12.0 s | `ro.boottime.zygote` |
| zygote preload (classes+resources) | 19.8 → 27.5 s | 7.7 s; framework-level trim only |
| PMS package scan (191 pkgs) | 31.4 → 35.7 s | 4.3 s |
| `ams_ready` | 44.4 s | late_start services begin (wpa_supplicant at ~44 s) |
| `enable_screen` | 47.7 s (pre-fix: 52.2) | nobootanimation gain |
| `boot_completed` | ~48 s | service polls every 1 s |
| `sf_stop_bootanim` / ready | 51.1 s (pre: 62.9) | nobootanimation gain |
| launcher interactive | ~55–60 s | Daijishō cold start dominates |
| screen alive (panel cycle) | ~66 s → ~52 s after v28 reorder | bounded by `input` needing system_server |

### Boot-speed levers applied

- `debug.sf.nobootanimation=1` in `/vendor/build.prop` — ~10 s off the visible
  tail plus CPU/GPU freed mid-boot.
- The service already avoids blocking: the rail latch runs before boot wait,
  the panel cycle waits for boot_completed (needs system_server for `input`).

### Levers left (performance phase)

- Package trim (191 packages → PMS scan 4.3 s; each removed app also saves RAM).
- Zygote preload trim is a framework change — high risk, skip unless needed.
- f2fs options are already good (`lazytime,noatime,flush_merge`).
- Consider `pm.dexopt` profiling for the launcher/game emulators at runtime.
