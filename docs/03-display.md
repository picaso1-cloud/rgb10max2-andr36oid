# 03 — Display

## Panel facts

- `elida,kd50t048a`, MIPI-DSI, **2 lanes**, 480×854@60 portrait.
- Init sequence and timing come from the stock go3 device tree; backlight is
  PWM with **251 brightness levels**, `vcc_backlight` 3.3 V rail.
- Android needs `androidboot.sf_orientation=ORIENTATION_270` on the kernel
  cmdline (via `boot.ini`) — the framebuffer/panel are portrait, the UI runs
  landscape.
- The DSI link negotiates `366 x 2 Mbps` when running.

## Kernel side (works, like stock)

At kernel boot (verified in pstore console log):

```
[0.49s] dw-mipi-dsi binds (after two -517 defers), panel-simple-dsi.0 binds
        (vsp/vsn missing → dummy regulators, harmless)
[0.53s] display-subsystem bound, fb0 registered, drm initialized
[11.4s] "Freeing drm_logo memory: 7168K" — the boot logo was composited
```

On stock RetroOZ the VOP enables the CRTC at **1.4 s** and the logo is visible
from power-on. The panel, regulators, DSI lanes and backlight are all fine.

## The Android black-screen bug

**Symptom:** after power-on the screen stays black through the whole Android
boot; the system is actually up (adb works, pressing power shows the shutdown
menu); picture appears only after pressing the power button (~2 min).

**Evidence chain (from a live boot, v26 image):**

1. Kernel logo composited (pstore, above) — but **no `vop_crtc_enable` /
   "Update mode" line anywhere in the boot console log**.
2. `boot_progress` events show the full boot proceeding normally; boot
   animation rendered frames (`sf_frame_dur: [BootAnimation#0,...]`).
3. `screencap` mid-boot shows the launcher **fully composed** — the compositor
   and GPU are fine; only the physical scanout is missing.
4. `DisplayManagerService` logs **no** display state change during the entire
   boot — Android believes the screen is ON from the start (brightness was
   even being set).
5. The **first** `Update mode to 480x854p60` appears only at the moment the
   user pressed the power button: `DisplayPowerController` did sleep→wake
   (DPMS OFF→ON), and the ON transition triggered the CRTC modeset.

**Root cause:** when SurfaceFlinger/HWC take over DRM master (logo freed at
~11.4 s), the CRTC is **never (re)enabled for the Android framebuffer**. The
legacy rockchip hwcomposer's presentation path doesn't force a modeset, and
since Android's power state says "ON" from the start, nothing else triggers one
either. Result: frames rendered into a framebuffer that nothing scans out.

**Why it matters per-boot:** this is a handoff bug, not a panel/driver failure —
so no DTS change fixes it. It needs either (a) a userspace DPMS cycle, or
(b) a kernel/HWC patch (kernel-build phase).

## The fix (v27): automatic DPMS cycle

In the boot service, right after `sys.boot_completed`:

```sh
# wait for bootanim to leave "running" (empty is fine: nobootanim builds)
while [ "$(getprop init.svc.bootanim)" = "running" ] && [ $M -lt 120 ]; do sleep 1; done
input keyevent KEYCODE_POWER     # sleep  → DisplayPowerController: OFF
sleep 1.5
input keyevent KEYCODE_POWER     # wake   → ON → CRTC modeset → picture
```

Verified on hardware: `Waking up from Asleep (reason=WAKE_REASON_POWER_BUTTON)`
→ `DisplayManagerService: "Built-in Screen", ON` → launcher visible, **no user
interaction**. Screen alive ~66 s after power-on (before the v28 reorder puts
it right after boot_completed ~52 s).

### Triggers that were tested and rejected

| Trigger | Result |
|---|---|
| `echo 1/0 > /sys/class/graphics/fb0/blank` | no DPMS on this kernel (no modeset, no effect) |
| `echo off/on > /sys/class/drm/card0-DSI-1/dpms` | sysfs node is **read-only** |
| `input keyevent KEYCODE_POWER` ×2 | ✅ the only working userspace trigger |

## Boot animation

`debug.sf.nobootanimation` ships as `0` in `/vendor/build.prop`. Setting it to
`1` (vendor is writable; **`/system` is erofs and is not**) saves the animation
tail and frees CPU/GPU during the heavy boot phase:

| Milestone | before | after |
|---|---|---|
| `boot_progress_enable_screen` | 52.2 s | 47.7 s |
| `sf_stop_bootanim` / system ready | 62.9 s | **51.1 s** |

The animation was invisible anyway (rendered into the dead panel), so nothing
is lost visually.

## Orientation

`androidboot.sf_orientation=ORIENTATION_270` in `boot.ini` is required. Without
it the UI is rotated 90° relative to the chassis.
