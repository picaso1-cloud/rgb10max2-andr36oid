# 06 — Android internals (the patterns that made the port work)

## The vendor service pattern

`/system` is **erofs** (read-only, no live edits possible). `/vendor` is ext4
and can be remounted rw **live on a running device** (no dm-verity on this
build). Everything the port needs therefore lives in vendor:

```
/vendor/etc/init/zz-diag.rc     # init hook
/vendor/etc/diag.sh             # the service (v27)
/vendor/etc/rk817tool           # static i2c tool
/vendor/etc/wifitool            # static nl80211 tool
/vendor/build.prop              # debug.sf.nobootanimation=1 lives here
```

`zz-diag.rc`:

```
on post-fs-data
    start andr36oid-diag

service andr36oid-diag /system/bin/sh /vendor/etc/diag.sh
    user root
    group root
    disabled
```

**Critical detail:** the service must be a **long-running daemon** (ends in a
`while true` loop). Init kills oneshot services' children; everything that
needs to stay alive (root daemon, watchdog) must run *inside* the service
process, not as its children.

SELinux on this build is **permissive** — vendor services can write sysfs,
inject input, remount, and i2c-transfer without denials blocking them (denial
*messages* still appear in the log).

## The root-command daemon (live control without USB adb)

Because the phy port is disabled (charging requirement), there is no cable adb.
The service main loop watches a FIFO-like pair in `/data/local/tmp`:

```sh
if [ -f /data/local/tmp/cmd.in ]; then
    mv /data/local/tmp/cmd.in /data/local/tmp/cmd.running
    /system/bin/sh /data/local/tmp/cmd.running > /data/local/tmp/cmd.out 2>&1
    chmod 666 /data/local/tmp/cmd.out
fi
```

Usage over TCP adb:

```sh
adb shell 'cat > /data/local/tmp/cmd.in << "EOF"
{ ...root commands... } > /data/local/tmp/cmd.out 2>&1
chmod 666 /data/local/tmp/cmd.out
EOF'
# ~1 s later:
adb shell 'cat /data/local/tmp/cmd.out'
```

This gives **root shell** on any boot where Wi-Fi works — it is how every live
diagnosis in this project was done. Files created by root need explicit
`chmod 666` before `adb pull` can fetch them (adb runs as shell, not root).

## adb over TCP

The service sets `service.adb.tcp.port 5555` and restarts adbd after
`boot_completed`. Until then the device shows `offline` in `adb devices` right
after boot — **wait ~40–50 s before declaring the network dead**.

```sh
adb connect <ip>:5555   # repeat until state "device"
```

## Boot analysis toolkit (what found the display bug)

| Source | What it gives | Caveat |
|---|---|---|
| `logcat -d -b events` | `boot_progress_*`, `sf_stop_bootanim` — ms since boot; small buffer survives the whole boot | — |
| `logcat -d -b system/main` | service init, DDM/PowerManager events | wraps fast (~2 min of heavy boot); `chatty` **expires lines** (uid=1000 system threads) — log-only conclusions need cross-checking |
| `dmesg` | kernel log | ring wraps after ~2 min of Android logging; early lines lost |
| `/sys/fs/pstore/console-ramoops-0` | **persistent** console from boot start | only covers ~first 12 s here (console loglevel drops after Android starts); copied to `/data/local/tmp` by the service |
| `screencap` | what the compositor sees | shows content even when the panel is black — perfect for "compositor vs scanout" discrimination |
| `/sys/class/drm/card0-DSI-1/{status,dpms}` | connector state (`connected`, `On`) | dpms **read-only** |
| `getprop ro.boottime.*` | exact init start time of every service | — |
| `logcat -d -b events | grep boot_progress` | the boot timeline | ms are since boot, not wall clock |

## Boot-speed levers (Android 11, 1 GB)

Applied:
- `debug.sf.nobootanimation=1` (`/vendor/build.prop`) — ~10 s visible-tail gain,
  CPU/GPU freed mid-boot.

Candidates for the performance phase:
- package trim (PMS scan 4.3 s @ 191 packages; RAM win too),
- disable unneeded Hal services in vendor rc (`thermal-mock`, `memtrack`, …),
- keep zram (helps on 1 GB),
- do **not** touch zygote preload without a full test cycle.

## Performance phase (Nov 2026)

Baseline (idle, charging, launcher up): 975 MB RAM / **369 MB available**;
51 init services; 191 packages; CPU schedutil 408–1296 MHz; GPU Mali-G31
`simple_ondemand` **policy-capped at 520 MHz** (600 MHz OPP exists!); SoC 62 °C.

Applied (all verified live):

| Change | Effect |
|---|---|
| `am kill-all` (drop cached helper apps) | +66 MB available instantly |
| `device_config put activity_manager max_cached_processes 12` | keeps cached-process count down (default 32) |
| disabled `vendor.thermal-hal-2-0-mock` (rename its rc) | fake-data HAL gone; real thermal is kernel-side (`soc-thermal`, `gpu-thermal` zones) |
| `echo 600000000 > /sys/class/devfreq/ff400000.gpu/max_freq` | GPU unlocked to the stock-firmware 600 MHz gaming clock (idle stays 520, ramps on load) |
| force-stop `org.videolan.vlc` PlaybackService | +18 MB PSS |

Result: **442 MB available** (was 369). GPU change needs a gameplay test
before being baked into the service.

### v28: the background janitor + runtime tuning (in the service, every boot)

Kernel sysctls reset each boot, so the service re-applies:

| Tuning | Value | Fallback |
|---|---|---|
| `vm.swappiness` | 100 (push anon pages into compressed zram) | 60 |
| SD `read_ahead_kb` | 512 (ROM streaming) | 128 |
| `queue/iostats` | 0 (less write overhead) | 1 |
| GPU `max_freq` | 600 MHz | 520000000 |
| UI animation scales | 0.5 | 1.0 |

**Janitor, every 40 s:** `am kill-all` (kills only cached processes — games,
the frontend, adb and foreground services are untouched) plus a full
`am force-stop` of Thorium/VLC **only when they are not the foreground
activity**. Syncthing is deliberately excluded — it must run in the background
to sync; stop it by hand when unwanted.

**Game mode toggle** (`/data/local/tmp/gamemode` / `gamemode-off`): pins the
CPU floor at 1.008 GHz so cores never ramp from idle mid-game; GPU already at
600 MHz max. Revert = floor back to 408 MHz.

**zram:** comp_algorithm shows `lzo lzo-rle [lz4]` — **already lz4** (the
bracketed algorithm is active). The right low-RAM lever is swappiness, not the
compressor.

### Edge-tech tips not yet applied (with fallbacks)

- **LMK tuning** via `/vendor/build.prop` (`ro.lmk.kill_heaviest_task=true`, …)
  — makes the out-of-memory killer take the biggest cached app first. Needs a
  reboot to test; remove the line if the launcher ever gets killed wrongly.
- **zram writeback** to SD (backing device) — complex, wears the card, skip.
- **CPU governor `performance` during games** — more heat (SoC already 62 °C
  idle-charging); the schedutil + 1 GHz floor in game mode is the better
  trade here.

App inventory (23 third-party): emulators (PPSSPP/Flycast/EasyRPG/RetroArch/
Mupen64+/Lemuroid), frontend (Daijishō), tools (Magisk, Droid-ify, NewPipe,
Limelight for streaming, Syncthing for ROM sync, VLC, Thorium browser —
341 MB of /data, the biggest item).

Boot-speed levers left: package trim (PMS scan 4.3 s @ 191 pkgs), vendor HAL
trim (`memtrack`, remaining mocks). Zygote preload trim is framework surgery —
skip. f2fs options already good (`lazytime,noatime,flush_merge`).

## Dual zygote / no GApps

The build keeps dual zygote (32+64) and ships no Google Apps — keep both
properties when rebasing; the joypad-only UI depends on the AOSP-derived
launcher stack (Daijishō is the actual frontend on top).
