# 02 — Power and charging (the hard-won model)

This is the most subtle subsystem on the board. Three wrong models were tested
before the correct one was proven. Read the **model** section first; the rest is
the evidence.

## The correct model (v26, verified)

1. **The power-port charge input and the OTG/hub 5 V net are ONE net.**
2. **The rk817 can charge from the input AND feed the net (hub) at the same
   time.** Charging and Wi-Fi together are physically possible — stock does it.
3. The only thing that ever broke charging was **software**: the OTG *phy port*
   telling the charger driver about OTG. That is the trap.

## The trap: phy port ↔ charger driver

When the u2phy OTG **port is enabled** (`status = "okay"`), the phy fires
`EXTCON_USB_VBUS_EN`. The rk817 charger driver reacts by setting `otg_in=1` and
**disabling its own plug/plug-out interrupts** (`disable_irq(charge->plugin_irq)`
— visible in the driver code and in our v13 charger logs). From that moment:

- the charger never notices a charger being plugged in,
- charging appears dead until reboot,
- the battery/AC state can go stale ("always charging" or "never charging").

**With the phy port disabled** (which is what the stock firmware ships), the
charger never hears about OTG, keeps its plug detection alive, and charges
normally — including while Wi-Fi runs.

### The DTS incantation (stock topology)

```
&u2phy_otg {
    status = "disabled";
};

&usb20_otg {
    dr_mode = "host";
    phys = <&u2phy_otg>;      /* keep the phandle! */
    phy-names = "usb2-phy";
};
```

Why keep `phys` pointing at a disabled port? The phy lookup **defers** until the
u2phy driver is up (UTMI clock running), then returns `-ENODEV`, so dwc2 probes
**phyless with its clock already on**. Removing the `phys` property outright
makes dwc2 probe too early and its core soft-reset **hangs**
(`dwc2_core_reset: HANG!`, probe fails with -16). This timing dance is load-bearing.

### Other DTS rules that matter for charging

- `dc_det_gpio` (gpio0 PB3) must **stay** as in the base tree. Deleting it
  (tried in v21) also disabled the charge path entirely.
- `dr_mode = "host"` on dwc2: Android's adb gadget switching (`init.usb.configfs.rc`)
  will fail on the missing phy — that's the documented trade-off (no cable ADB),
  it does not affect charging or Wi-Fi.

## The rk817 OTG rail latch

The hub's 5 V is fed by the rk817's OTG boost rail. On this board that rail
behaves as a **PMIC-side latch**:

- **Kernel writes never reach it.** Writing the OTG enable via the charger,
  the regulator, or the phy's `vbus-supply` all fail silently: the kernel
  register *cache* reads back "on" (`reg_b4=47 otg5v=1`) while the raw hardware
  register reads `0x06/0x07` and no power appears on the net.
- **What works: a raw i2c write `0xB4 = 0x47`** from userspace, after the
  charger driver is bound (the tool: [`tools/rk817tool.c`](../tools/rk817tool.c)).
- The latch **survives warm reboots**; a **cold power-off resets it**. So the
  boot service writes it on every cold boot.
- Writes while external 5 V is already present are harmless but unnecessary —
  with a charger plugged at boot, its 5 V feeds the shared net and the hub
  enumerates before the latch is even needed (`hub already up (devs=8)`).
- Latch armed ⇒ the PMIC sees its own output on the net ⇒ `dc_det` reads
  "present". This is a **self-sensing loop**, not a fault: with the phy port
  disabled it is cosmetic only (see below for the battery-icon caveat).
- The latch **cannot be cleared** by register writes while running — plugging a
  charger *after* arming the latch on battery needs a reboot (cold-reset) to
  charge. Documented limitation of v26/v27.

## The battery-icon caveat

With the latch armed on battery, `dc_det` reads "present" (self-sensing), so the
charger driver may report `ac online=1` / "Charging" with no charger. The fuel
gauge (battery %) is a separate driver and stays honest. A clean kernel-driver
patch (mask detection while the OTG rail is active) is the eventual fix —
deferred to the kernel-build phase. With a real charger in the power port,
`+869 mA` charging with `wlan0` connected was measured — charging truly works.

## The final service logic (v27)

[`service/diag.sh`](../service/diag.sh), started from `post-fs-data` via
[`service/zz-diag.rc`](../service/zz-diag.rc) (root, daemonized because init
kills oneshot children):

1. `insmod rtl8xxxu` (so the dongle binds the moment it enumerates)
2. pin runtime PM on the dwc2 platform device and usb buses
3. arm the latch (`rk817tool b4 47`) unless the hub is already up; on failure
   unbind/rebind `rk817-charger` and retry
4. after boot: Wi-Fi power-save off + `MA` regdomain (`wifitool`), adb over TCP
5. **panel DPMS cycle** (see [03-display.md](03-display.md))
6. main loop: root-command daemon + 30 s watchdog that re-arms the latch if the
   hub ever loses power

## Power state matrix (v27, all verified)

| Situation | Behavior |
|---|---|
| Console off, charger in power port | ✅ charges (PMIC autonomous) |
| Boot with charger in power port | ✅ charging + Wi-Fi (charger feeds the net, latch not needed) |
| Boot on battery | ✅ latch armed → Wi-Fi |
| Unplug charger while running | ✅ nothing changes (charger keeps plug detect; on battery the latch was already armed or the watchdog arms it) |
| Plug charger after latching on battery | ⚠️ needs a power cycle to charge (latch can't be cleared live) |
| Charger in the OTG port | ❌ hub/Wi-Fi killed by PMIC OTG-priority — wrong port by design |

## Failed models (do not re-try these)

- **v21/v23:** delete `dc_det_gpio` → indicator honest but *charging itself disabled*.
- **v24:** "charger-aware" service (skip latch when `dc_det=1`) — built on the
  false premise that the power-port input and the hub net were separate; they
  are one net.
- **v25:** unbind the charger driver while latched to hide the false icon —
  works cosmetically but hides real charging state; the real fix was the phy
  port (v26).
