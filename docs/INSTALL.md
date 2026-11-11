# Install — build and flash the RGB10 Max 2 image

## 1. What you need

- A **Powkiddy RGB10 Max 2** (`hwrev=v10-go3`, RK3326).
- A **microSD card**, 8 GB or larger. Prefer a genuine A1/A2 card; the flasher
  verifies the card after writing and will tell you if it corrupts data.
- A **USB card reader** (not the console itself).
- An **official andr36oid RK3326 image** (`.img` or `.img.xz`) — used as the base
  you are patching.

## 2. Build the image (Linux)

Dependencies: `bash`, `xz`, `fdisk`, `losetup`, `mount`, `e2fsprogs`,
`mtools`, `dosfstools`.

```sh
# Debian/Ubuntu
sudo apt install xz-utils fdisk mount e2fsprogs mtools dosfstools

# Arch
sudo pacman -S xz util-linux e2fsprogs mtools dosfstools
```

Build:

```sh
./scripts/build-rgb10max2-image.sh /path/to/andr36oid-rk3326.img.xz \
    -o RGB10MAX2-andr36oid.img
```

Or let it download the latest official release:

```sh
./scripts/build-rgb10max2-image.sh --download -o RGB10MAX2-andr36oid.img
```

The script mounts the base image's `BOOT` and `vendor` partitions read/write,
applies the RGB10 Max 2 profile, service and tools, and verifies the changes
inside the finished image. It also writes `RGB10MAX2-andr36oid.img.sha256`.

What it changes:

| Partition | Content |
|---|---|
| `BOOT` (p1) | `Devices/RGB10MAX2/` profile (DTBs, logo), `andr36oid-profiles.txt` line, `boot.ini` orientation + USB autosuspend |
| `vendor` (p8) | `diag.sh`, `zz-diag.rc`, `rk817tool`, `wifitool`, `debug.sf.nobootanimation=1` |

## 3. Flash the card

### Our flasher (recommended)

```sh
cd flasher
./build.sh                       # produces dist/ binaries, or use the releases
sudo ./dist/rgb10max2-flasher-linux-amd64 list
sudo ./dist/rgb10max2-flasher-linux-amd64 flash ../RGB10MAX2-andr36oid.img
```

It asks you to pick the card and type `YES`, writes the image, then **reads the
whole image back and compares its SHA-256**. A mismatch means the card is bad —
try another card rather than booting a corrupted install.

Options:

```
--device /dev/sdX   skip the picker
--yes               no confirmation prompt (use with care)
--no-verify         skip the read-back (not recommended)
```

### Official andr36oid flasher (alternative)

The official GUI flasher accepts a local image when you choose to provide one
manually. Use that if you prefer Windows and do not want to test our
experimental Windows writer.

## 4. First boot

1. Insert the card into the console and power on.
2. If the console is not auto-detected, choose **RGB10 Max 2** in the profile
   picker (the port adds `Devices/RGB10MAX2`).
3. The display should come up in the correct orientation. If it is rotated,
   the `boot.ini` orientation line did not reach the card — rebuild and reflash.
4. Wi-Fi should be available from battery on a cold boot. If it is not, see
   `docs/02-power-and-charging.md` (OTG latch / rk817tool).
5. ADB over Wi-Fi listens on port 5555 once the boot service has run.

## 5. Migrating an existing install

If you already had an andr36oid install with settings you want to keep:

```sh
./scripts/migrate-console-state.sh --adb <console-ip>
```

It moves per-install state (Tailscale auth and similar) to the new card. See the
script header for options.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `no removable devices found` | Card reader not attached, or the card is not a removable device. Run as root on Linux. |
| `could not unmount /dev/sdX1` | A file manager has the card open. Close it, or unmount manually. |
| `read-back mismatch` | The card corrupted data during writing. Replace the card — do not boot it. |
| Image does not fit | Card smaller than the image; use a larger card. |
| Black screen after the logo | `boot.ini` orientation/autosuspend edits missing; rebuild from a clean base. |
| Wi-Fi missing until reboot | OTG latch not flipped; check that `diag.sh` + `rk817tool` reached `vendor`. |
| Windows writer fails | Run as Administrator; the experimental writer uses `diskpart`. The official flasher is a safe fallback. |
