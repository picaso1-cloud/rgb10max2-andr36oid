#!/bin/sh
# Apply the RGB10 Max 2 key remap overlay to a ROCKNIX RK3326 card.
#
# usage: apply-rocknix-max2.sh /dev/sdX1
#
# The card's first partition (FAT, label ROCKNIX) contains extlinux/ and
# overlays/. This script backs up extlinux.conf, installs the .dtbo and adds
# it to the FDTOVERLAYS line. It does not touch the DTB, kernel or rootfs.
set -eu

PART="${1:-}"
[ -n "$PART" ] || { echo "usage: $0 /dev/sdX1   (the FAT ROCKNIX partition)"; exit 2; }
[ -b "$PART" ] || { echo "not a block device: $PART"; exit 2; }

HERE=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
DTBO="$HERE/rk3326-rgb10max2-keys.dtbo"
[ -f "$DTBO" ] || { echo "missing $DTBO (compile the .dts first)"; exit 2; }

MNT=$(mktemp -d)
cleanup() { umount "$MNT" 2>/dev/null || true; rmdir "$MNT" 2>/dev/null || true; }
trap cleanup EXIT INT TERM

mount "$PART" "$MNT"

if [ ! -f "$MNT/extlinux/extlinux.conf" ]; then
  echo "ERROR: $PART does not look like a ROCKNIX boot partition (no extlinux/extlinux.conf)"
  exit 1
fi

mkdir -p "$MNT/overlays"
cp -f "$DTBO" "$MNT/overlays/rgb10max2-keys.dtbo"

CONF="$MNT/extlinux/extlinux.conf"
cp -f "$CONF" "$CONF.bak-rgb10max2"

if grep -q "rgb10max2-keys.dtbo" "$CONF"; then
  echo "overlay already present in extlinux.conf"
elif grep -q "^[[:space:]]*FDTOVERLAYS" "$CONF"; then
  sed -i 's#^\([[:space:]]*FDTOVERLAYS.*\)$#\1 /overlays/rgb10max2-keys.dtbo#' "$CONF"
  echo "added overlay to existing FDTOVERLAYS line"
else
  # insert after the FDT/FDTDIR line, or before APPEND
  if grep -q "^[[:space:]]*FDT" "$CONF"; then
    sed -i '/^[[:space:]]*FDT/a\  FDTOVERLAYS /overlays/rgb10max2-keys.dtbo' "$CONF"
  else
    sed -i '/^[[:space:]]*APPEND/i\  FDTOVERLAYS /overlays/rgb10max2-keys.dtbo' "$CONF"
  fi
  echo "added a new FDTOVERLAYS line"
fi

sync
echo "--- extlinux.conf:"
grep -E "FDT|APPEND" "$CONF"
echo "done. remove the card safely."
