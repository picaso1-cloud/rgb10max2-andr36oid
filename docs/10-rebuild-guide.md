# 10 — Rebuilding the SD card (from scratch or clone)

The first card used for this port was a slow test card **that ended up dying**
(unstable reads — identical sectors returned different data between reads, and
its BOOT FAT corrupted: lost top-level files, I/O errors on directories). This
page documents the full rebuild so a fresh card reproduces everything.

## Card shopping

- **A2-class** microSD (the A rating = random-I/O performance; it is what makes
  Android usable). SanDisk Extreme/Extreme Pro, Samsung EVO Plus/Pro Plus,
  Kingston Canvas React Plus.
- **U3 / V30** minimum sequential speed.
- **64–128 GB** if you plan a ROM collection; the stock userdata partition is
  4.8 GB (extendable — see below).
- Avoid marketplace no-name "A2" cards.

## What a rebuild needs — two paths

### Path A: clone (identical copy of a healthy card)

If the current card is HEALTHY (verify: two reads of the same region must
match), a plain `dd` clone carries everything — apps, data, tailscale auth,
Wi-Fi, all patches:

```sh
dd if=/dev/mmcblk0 of=master.img bs=4M conv=fsync status=none
md5sum master.img
# write master.img to the new card, then extend userdata (below)
```

**Do NOT clone a card that shows read instability** — the corruption gets
copied. Use Path B.

### Path B: clean rebuild from the official image (used for the real rebuild)

1. **Base**: official release `andr36oid-*-rk3326.img.xz` from
   [andr36oid/releases](https://github.com/andr36oid/releases).
2. **Build** the master (applies every patch of this port):
   ```sh
   image-patcher/build-rgb10max2-image.sh <base.img.xz> -o master.img
   ```
   This installs: RGB10MAX2 boot profile (v31+ DTB: stick mux remap, L3/R3
   codes), patched boot.ini (orientation + USB autosuspend), profiles line,
   vendor service (v32: latch, panel cycle, janitor, tailscale autostart),
   rk817tool/wifitool, `debug.sf.nobootanimation=1`.
   The BOOT partition is rebuilt **with mtools** — kernel vfat writes proved
   flaky against sparse loop backing (a corrupted Devices/RGB10MAX2 directory
   was the failure mode; mtools is deterministic).
3. **Flash** master.img to the new card (Rufus/Etcher/dd).
4. **Extend userdata** to fill the bigger card (see below).
5. **Restore state** (see below).
6. Boot, connect Wi-Fi, done — no Tailscale re-login, apps intact.

### Extending userdata to fill the card

The image's GPT/PMBR references the original card size; after flashing a
bigger card, fix and grow:

```sh
DISK=/dev/mmcblk0   # the NEW card in a reader!
sgdisk -e $DISK                        # move GPT backup to real end
sgdisk -d 6 -n 6:0:0 -t 6:8300 $DISK   # delete+recreate p6 to fill card (keep order!)
partprobe $DISK
# mkfs.f2fs fresh, then restore the userdata backup:
mkfs.f2fs -l userdata $DISKp6
mount -t f2fs $DISKp6 /mnt/ud && tar xf userdata-full.tar -C /mnt/ud && sync && umount /mnt/ud
```

`userdata-full.tar` is a full tar of the previous card's `/data` (taken before
retiring the card). It carries all apps (`/data/app`), their data (Daijishō
library, Moonlight hosts, settings), Wi-Fi networks, and
`/data/local/tmp/ts/state` (Tailscale authentication — **no re-login needed**).

Minimal alternative (no apps/data to keep): only restore the auth file —

```sh
tar xzf essentials.tgz -C /mnt/ud/local/tmp/   # ts/state/tailscaled.state
```

(`essentials.tgz` = just `ts/state/tailscaled.state`; pulled from a running
console via the root daemon. Without it, the next `tailscale up` prints a
login URL to re-authenticate — a 2-minute step.)

### Verify after flash (before booting the console)

- `fdisk -l` on the card: p1 starts at **32768**, p8 exists.
- mtools: `mdir -i card@@16777216 ::/boot.ini` works and contains
  `ORIENTATION_270`; `::/rk3326-r36s-android.dtb` pulls a file whose md5 is
  the production DTB (`75d0c49c…`).
- Kernel mount quirk: verify with **mtools**, not the kernel vfat driver —
  loop-device page-cache staleness made correct images *look* corrupted.

## Console-first-boot checklist (new card)

1. Boots to logo → black ~45 s → screen self-on (panel cycle), adb TCP on.
2. `adb connect <ip>:5555`, check `diag-status.txt` (latch armed, tailscale
   started, boot time).
3. `tailscale status` shows the tailnet, IP `100.x.x.x` (or the migrated
   state's IP).
4. Mouse mode (hold Select+Start), right stick = cursor, all four directions
   correct.
5. Wi-Fi networks should already be in the list (from the userdata restore).
