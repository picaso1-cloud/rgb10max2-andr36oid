#!/bin/bash
# build-dtb.sh — recompile the RGB10 Max 2 device tree from source.
#
# Needs the andr36oid r36s kernel headers (arch/arm64/boot/dts/rockchip +
# include/dt-bindings + uapi). They are fetched with a blobless sparse clone,
# which is small (~20 MB).
#
# Usage: ./build-dtb.sh            # rebuild and install the DTB
#        KWORK=/path ./build-dtb.sh
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
WORK="${KWORK:-/tmp/rgb10max2-kdts}"
KREPO="https://github.com/andr36oid/android_kernel_gameconsole_r36s"
OUT="$HERE/../device/boot-profile/rk3326-r36s-android.dtb"

need() { command -v "$1" >/dev/null || { echo "missing: $1" >&2; exit 1; }; }
need git; need cpp; need dtc

if [ ! -d "$WORK/arch/arm64/boot/dts/rockchip" ]; then
    echo "fetching kernel headers into $WORK ..."
    rm -rf "$WORK"
    git clone --depth 1 --filter=blob:none --sparse "$KREPO" "$WORK"
    git -C "$WORK" sparse-checkout set arch/arm64/boot/dts/rockchip include/dt-bindings
    git -C "$WORK" sparse-checkout add include/uapi arch/arm64/include/uapi
fi

echo "preprocessing ..."
cpp -nostdinc -I"$WORK" -I"$WORK/include" -I"$WORK/include/uapi" \
    -I"$WORK/arch/arm64/include" -I"$WORK/arch/arm64/include/uapi" \
    -I"$WORK/arch/arm64/boot/dts" -I"$WORK/arch/arm64/boot/dts/rockchip" \
    -undef -D__DTS__ -x assembler-with-cpp \
    -o "$WORK/rgb10max2.pre.dts" "$HERE/rk3326-rgb10max2-android.dts"

echo "compiling ..."
dtc -I dts -O dtb -o "$WORK/rgb10max2.dtb" "$WORK/rgb10max2.pre.dts" 2>/dev/null

cp "$WORK/rgb10max2.dtb" "$OUT"
echo "installed $OUT"
sha256sum "$OUT"
