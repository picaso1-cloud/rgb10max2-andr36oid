# 05 — Wi-Fi

## Hardware

- **RTL8723BU** USB combo dongle (Wi-Fi 2.4 GHz + BT 4.0) behind the internal
  hub on the dwc2 bus. USB ID `0bda:b720`.
- Driver: **`rtl8xxxu`** (mainline). The vendor driver (`8723bu` out-of-tree)
  is not used.
- Physical power switch on the chassis cuts the module from the bus — software
  must tolerate the module disappearing/reappearing.
- The module's power comes from the shared 5 V net ⇒ **Wi-Fi availability is
  tied to the OTG rail latch** (see [02-power-and-charging.md](02-power-and-charging.md)).

## Making it work

1. **Preload the driver** before/at enumeration: `insmod /vendor/lib/modules/rtl8xxxu.ko`
   in the boot service — otherwise the dongle binds late or not at all.
2. If the hub has no power (latch not armed), nothing appears on the bus; the
   service arms the latch first, the hub then enumerates (`usbdevs=8`).
3. `wlan0` appears ~10–20 s after the hub has power; `wpa_supplicant` itself
   starts with `late_start` (~44 s into boot) — that's framework scheduling,
   not the driver.

## The quirks that matter

| Quirk | Detail | Workaround |
|---|---|---|
| No WPA3/SAE/PMF | `rtl8xxxu` lacks SAE and MFP | connect to **WPA2-only** PSK networks; on mixed APs force it: `cmd wifi connect-network <SSID> wpa2 <pass>` |
| 2.4 GHz only | by hardware | — |
| Weak links drop | AP kicks the client with "missing ACKs" (reason=34) below ≈ **−75 dBm** | position/antenna; keep power save off |
| Power save hurts | default PS adds latency + drops | **disable it** (below), re-apply periodically |
| No `iw` binary | image ships no iw | use [`tools/wifitool.c`](../tools/wifitool.c) (raw nl80211, static-linked) |

## Radio tuning (baked into the service)

```sh
wifitool ps-off    # disable 802.11 power save (re-applied every 30 s)
wifitool reg MA    # regulatory domain; `iw reg set` unavailable (no iw)
```

`ps-off` is re-applied by the service watchdog every 30 s because the stack can
re-enable it after reassociation. With regdomain MA and PS off, throughput and
latency are stable enough for game streaming.

## Useful live commands (via adb / the root daemon)

```sh
cmd wifi list-scan-results          # note: partial scans while connected
cmd wifi connect-network <SSID> wpa2 <pass>
ip addr show wlan0
cat /proc/net/wireless              # signal level
```

## Bluetooth

Same combo module exposes `hci0` (BT over USB). Verified working; no special
handling beyond the hub needing power.

## Known-good connectivity tests

- Mixed WPA/WPA2 (TKIP) network: connects (forcing the WPA2 config helps it
  pick the right AKM).
- WPA3-only / PMF-required: does **not** connect (driver limitation).
- Console seen at `192.168.1.50` (HOME network) and `192.168.1.51` (other
  SSID) — it switches networks cleanly when the stored config is forced.
