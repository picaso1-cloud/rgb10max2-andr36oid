# 07 — Methods and tools (how to work on this console)

## SD card surgery rules (learned the hard way)

**Reading a card in a reader (host side):**

1. Always `umount` everything first, then remount fresh — **stale/ghost mounts
   lie** (old directory listings after the card was rewritten).
2. After reader dropouts, the SD device may need a rescan:
   `partprobe /dev/mmcblk0` or unbind/rebind the mmc host driver
   (`echo mmc0:b368 > /sys/bus/mmc/drivers/mmcblk/unbind` … not needed if you
   unplug/replug).
3. Sanity-check the card identity before any write:
   `cat /sys/block/mmcblk0/mmcblk0p1/start` must print `32768`.
4. **Never `dd` without `[ -b /dev/mmcblk0 ]`** — a typo once created a 5 GB
   *regular file named* `/dev/mmcblk0` instead of writing the card.
5. For partition files use the sys-provided major:minor:

```sh
DEV=$(cat /sys/block/mmcblk0/mmcblk0p8/dev)
mknod /dev/mmcblk0p8 b ${DEV%:*} ${DEV#*:}
mount /dev/mmcblk0p8 /mnt/vendorpart
```

6. `echo 3 > /proc/sys/vm/drop_caches` before verification reads; verify with
   `cmp` (byte compare) after every write, and `sync` before unmount.

**Live card work (card inside the console):** `/vendor` remounts rw over adb
(see [06-android-internals.md](06-android-internals.md)); BOOT and `/system`
changes need the card in a reader.

## Producing a full image from a card

```sh
dd if=/dev/mmcblk0 of=andr36oid-rgb10max2-vNN.img bs=4M count=1210 conv=fsync status=none
```

(covers through p8 + GPT backup area; the image is ~4.73 GiB). Verify by
loop-mounting p1 (BOOT, `offset=16777216`) and p8 (vendor,
`offset=285212672`) read-only and `cmp`-ing against the working files.

## DTB workflow

See [04-boot-chain.md](04-boot-chain.md) for the sparse-checkout + cpp + dtc
flow and the `fdtget` verification triple. The compiled DTB must be byte-copied
to **both** `BOOT/rk3326-r36s-android.dtb` and `BOOT/Devices/RGB10MAX2/`.

## The two static tools

### `tools/rk817tool.c`

Reads/writes rk817 registers over `/dev/i2c-0` (address 0x20). The only thing
that can flip the OTG rail latch:

```sh
rk817tool b4 47        # arm the OTG rail latch (hub power)
rk817tool b4           # read the raw hardware register (truth)
```

Why raw i2c: every kernel-side path (charger, regulator, phy vbus-supply)
writes a **cached** register view and never reaches the hardware — reads lie
(`reg_b4=47` while the raw register is `0x07`). Cross-compile static for
aarch64 and drop at `/vendor/etc/rk817tool`.

### `tools/wifitool.c`

Raw nl80211 (no libnl dependency, static-linked):

```sh
wifitool ps-off       # disable 802.11 power save on wlan0
wifitool reg MA       # set regulatory domain (image ships no iw)
```

## Debugging flows that paid off

1. **"Is it dead or just blind?"** — `screencap` over adb: content ⇒ compositor
   fine, panel/scanout problem; no content ⇒ app/compositor problem.
2. **"What did the kernel do at boot?"** — pstore console (`console-ramoops-0`)
   for the first seconds; `dmesg` only for the recent minutes.
3. **"When did the display first enable?"** — `dmesg | grep "Update mode"` right
   after boot (the ring still holds it early; it wraps later).
4. **"Why is boot slow?"** — `logcat -d -b events | grep boot_progress` (ms
   since boot) + `getprop | grep ro.boottime | sort -t= -k2 -rn` for the slowest
   services.
5. **"Is the charger really charging?"** — read
   `/sys/class/power_supply/battery/{status,voltage_now,current_now}`:
   `Charging 3.899 V +869 mA` is real charging, not just a flag.

## Version numbering

Every validated change bumps the version: DTS revisions and service revisions
move together (vNN tag = the pair), with an entry in
[08-validation-log.md](08-validation-log.md) capturing what the version proved.
Production images are named `andr36oid-rgb10max2-vNN.img` with an md5 in the
release notes.
