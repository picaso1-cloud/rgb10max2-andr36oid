# DTS compile validation — 2026-11-05

## Method
- Toolchain: host cpp (preprocessor) + dtc, against the real andr36oid kernel
  include tree (`android_kernel_gameconsole_r36s` @ lineage-18.1, shallow sparse
  clone at /tmp/opencode/kernel-src).
- Control: `rk3326-r46h-android.dts` (upstream known-good) compiled with the
  exact same pipeline. Both compile clean.

## Results
| Check | Expected | Compiled DTB | Verdict |
|---|---|---|---|
| panel compatible | elida,kd50t048a | ✓ | OK |
| dsi,lanes | 2 (go3 panel!) | 2 | OK |
| dsi,flags | VIDEO(1)+BURST(2)+LPM(0x800) = 0x803 | 0x803 | OK (R46H control = 0xa03, has EOT) |
| panel reset | gpio3 PC0 active-low (kern.log gpio 112) | <gpio3 16 1> | OK |
| timings | 480x854 @ 27.5 MHz, hfp2 hs10 hbp2 vfp12 vs2 vbp60 | ✓ | OK |
| backlight | 251 levels, default 125, pwm1 40000ns | ✓ | OK |
| joypad | invert-absx/absy (base) + invert-absrx + missing fn | ✓ | OK |
| battery | 4000 mAh (0xfa0), qmax 4020 (0xfb4), go3 OCV table | ✓ | OK (table verified hex→dec by script) |
| charger | 2000 mA / 4200 mV / finish 300, bat_low_gpio deleted | ✓ | OK |
| leds | heartbeat kept, charge gpio0 PB5 added, vibrator deleted | ✓ (0 vibrator refs) | OK |
| second card slot | disabled like R46H | ✓ | OK |

## Files
- Source: `device-work/kernel/rk3326-rgb10max2-android.dts`
- Compiled: `device-work/kernel/rk3326-rgb10max2-android.dtb` (94,698 bytes)
- Tooling note: cpp needs `-I <kernel>/include` (not relative); the sparse clone
  lazily fetches dt-bindings blobs — force with
  `git checkout HEAD -- include/dt-bindings` and add `include/uapi/linux`
  (dt-bindings/input/linux-event-codes.h is a symlink to it).

## Next (kernel side)
- This DTB is drop-in for the andr36oid kernel build: add to
  `arch/arm64/boot/dts/rockchip/Makefile`:
  `dtb-$(CONFIG_ARCH_ROCKCHIP) += rk3326-rgb10max2-android.dtb`
- Optional early smoke test on RetroOZ: compile the same DTS against the
  RetroOZ kernel tree (same 4.4 family), swap the DTB in the BOOT partition,
  confirm panel/input/LED before any Android build.

## v12 (2026-11-05, after reference-image study) — HOST PAIR ENABLED
**Discovery chain:**
- Studied local reference: 351droid LineageOS 18.1 for RGB10 Max 2021 (kernel 4.19.172).
  Its DTB: otg-port DISABLED, host-port okay; OTG-as-host worked via u-boot hwrev fixups.
- andr36oid's own rk3326-odroidgo3-linux.dts (our board class, ArkOS-working Wi-Fi):
  otg-port DISABLED, host-port okay.
- r36s-android.dtsi (our base): otg-port okay + `rockchip,vbus-always-on` +
  `rockchip,no-vbus-en-extcon` — the dtsi comment states the RK817 charger turns
  otg_switch off at probe and "never ask the charger to enable its OTG boost".
  => otg_switch-based fixes (v9..v11) could never stick; wrong bus AND deliberately
  suppressed extcon event.
- rk817_charger.c confirmed: probe writes OTG_EN=DISABLE to the PMIC directly
  (line ~1632); 5V returns only via EXTCON_USB_VBUS_EN event (suppressed!).
- In ALL previous boots usb_host0_ehci@ff340000 / usb_host0_ohci@ff350000 were
  status=disabled — the bus the hub almost certainly hangs on was never tested.

**Change:** `&usb_host0_ehci`/`&usb_host0_ohci` status okay (host-port already okay).
Everything else identical to v9/v11 verified state.

**Verified fdtget:** ehci=okay, ohci=okay, host-port=okay, otg-port=okay,
dwc2 dr_mode=host, panel 2 lanes, buttons sw1..sw14 = 103/108/105/106/304/305/308/307/315/314/708/709/312/313.

## v13 (2026-11-05, after RetroOZ/4.4 chain analysis) — THE RAIL OWNERSHIP FIX
**Root cause (fully traced):** base dtsi sets `rockchip,no-vbus-en-extcon` on
u2phy_otg → phy never fires EXTCON_USB_VBUS_EN → rk817 charger (whose probe
does an unconditional OTG_EN=DISABLE field write) never re-enables otg_switch
→ 5V to the Genesys hub dead in every boot.
Cross-checks: 4.4 (RetroOZ) dwc2 treats phy-get -ENODEV as optional (phyless,
u-boot-prepared port, charger+phy never touched VBUS) — that's why RetroOZ
worked. 4.19 dwc2 also tolerates -ENODEV (backup path if extcon fails).
regulator core: devm_regulator_get(dev,"otg_switch") resolves by NAME fallback
(regulator_lookup_by_name) — charger has a real handle.
**Changes vs v12:**
- `/delete-property/ rockchip,no-vbus-en-extcon` on u2phy_otg
- removed `regulator-always-on` from otg_switch (framework no-ops vs the
  charger's direct register write = fake success; charger must own the rail)
- removed `vusb_a-supply` from dwc2 (dwc2 runtime PM must never cut hub power;
  matches the reported "wifi icon dies after ~20-30s" symptom)
- kept: dr_mode=host, vbus-supply on phy, host pair enabled, buttons/panel
**Diag:** charger-proof.txt captures "asks for OTG 5 V"/"OTG 5 V on" dmesg
lines + otg_switch/boost regulator states after the role toggle.
**Verified fdtget:** suppression gone, vbus-supply present (0x7f), always-on
gone, vusb_a gone, ehci/ohci okay, dr_mode=host, buttons 315/314, lanes 2.

## v15 (2026-11-06) — stock-replica phyless with correct probe timing
**v14 post-mortem (from report dmesg):** dwc2 probe FAILED —
"dwc2_core_reset: HANG! Soft Reset timeout" at 0.26s, error -16. Deleting the
phys property made dwc2 probe BEFORE the u2phy driver enabled the UTMI clock
(0.43s) → reset timeout → dwc2 never came up (the "usb1" in the watch was
OHCI). The phyless CONCEPT was right; the timing was fatal. Stock 4.4 probed
dwc2 at 9.5s, long after the phy clock was on.
**v15 = exact stock recipe on 4.19:**
- u2phy_otg status=disabled (phy driver skips the port; UTMI keeps u-boot state)
- dwc2 keeps phys = <&u2phy_otg> + phy-names: phy lookup defers until the u2phy
  provider registers (clock on), then of_phy_simple_xlate returns -ENODEV for
  the unregistered port → dwc2 probes PHYLESS WITH CLOCK ON (verified in
  phy-core.c: of_phy_provider_lookup + of_phy_simple_xlate)
- vusb_a-supply = <&otg_switch> on dwc2: dwc2's lowlevel init enables it at its
  deferred probe (~0.44s) — a real regulator write that overrides the charger's
  probe-time kill (0.33s). Rail timeline: u-boot ON → charger kill 0.33s →
  dwc2 re-enable 0.44s → hub powers + connects ~0.5s+
- diag: dwc2/ehci/ohci runtime PM pinned ON (power/control=on) so nothing idles
  the rail off; adbd TCP armed unconditionally (phone-tethering ADB path)
**Verified fdtget:** otg-port disabled, phys=0x80, vusb_a=0x81, dr_mode=host,
ehci/ohci okay, buttons 315/314, lanes 2.

## v17 (2026-11-06) — the rail lifecycle test (post live-ADB breakthrough)
**Live findings (adb over Wi-Fi, USB-C OTG cable to PC):**
- PC's 5V VBUS on the OTG net → hub+dongle enumerate in ~3s, Wi-Fi works perfectly
- Unplug → dead in <3s; replug → back in ~3s; otg_switch=disabled throughout
- Working-state full logs captured: diag-live-working/ (dmesg+logcat)
- Charger idle state: otg5v=0, reg_b4=0x63, boost_only_for_otg=0
**v17 = v13 extcon chain on v16 base** (port enabled, suppression deleted,
phy vbus-supply, managed dwc2, autosuspend off, PM pins) + diag that toggles
the charger event early and samples reg_b4/otg5v/rail/devs every 0.5s for 3
min. Cable-OUT boot decides: hub up = fixed; register on + hub dead = the
board never wires otg_switch to the OTG net (hardware; workaround = 5V on
the OTG port); register re-killed = software killer to patch.

## FINAL MECHANISM (2026-11-06, live adb + register tooling)
**The RK817 OTG rail is a PMIC-side LATCH, not a normal register:**
- Kernel driver writes (charger OTG_EN, regulator enable, phy vbus-supply) NEVER
  reach it — regmap cache claims "on" (reg_b4=47/otg5v=1) while raw i2c reads 07
- One RAW i2c write of 0xB4=0x47 (with the charger driver unbound) flips the
  latch: hub powers from battery, Wi-Fi works standalone
- The latch SURVIVES warm reboots (PMIC stays powered) but resets on cold
  power-off -> must be re-armed at each cold boot
- Writing while external 5V is on the OTG net = collision -> PMIC cuts power
  (therefore the boot ladder only pokes when the hub is NOT enumerated)
- Charger driver can stay bound in normal daily state (latch unaffected once set);
  unbind is only the fallback if a cold-boot poke doesn't take
**Production v20 diag:** insmod rtl8xxxu early -> if hub absent: poke b4=47 ->
verify -> fallback unbind/poke/rebind ladder -> ADB TCP + root daemon; OTG port
left to normal data roles (no role toggling).

## v21/v22 (2026-11-06) — production polish
- v21 DTB: /delete-property/ dc_det_gpio on rk817 charger node -> no false
  "charging" (DC detect pin floats high on this board; R36S base dtsi wiring).
  USB-C charging unaffected.
- v22 diag: radio tuning baked in — wifitool (static nl80211 helper, no libnl)
  sets power_save off + reg MA once wlan0 exists, plus a 30s watchdog (power
  save state resets on driver reload). Tools installed to /vendor/etc
  (rk817tool, wifitool) with /data/local/tmp fallback.
- Wi-Fi robustness finding: HOME-NETWORK (WPA2+TKIP mixed) associates fine;
  disconnects were reason=34 (missing ACKs) at -76 dBm; connects solidly at
  -62 dBm. TKIP is not the blocker; signal/ACK robustness is.

## v24 (2026-11-07) — charger-aware rail (final power design)
**Findings:**
- charging while the console is OFF works (PMIC autonomous) — confirmed by the user
- the dc_det gpio (gpio0 PB3) senses the shared VBUS net: a real charger OR the
  latched OTG rail both read "present". v21's removal of dc_det disabled the
  charge path too, so charging while running was broken and the indicator was
  only "fixed" by disabling detection
- the latch cannot be cleared by register writes (b4 writes not stored; only a
  cold power-off resets it)
- charger on the OTG port kills the hub (PMIC prioritises input, OTG output off)
**v24 design:**
- dc_det_gpio restored (stock ACTIVE_HIGH)
- service reads dc_det_pin at boot: 1 = charger -> skip latch (charging works,
  Wi-Fi off); 0 = battery -> poke latch (Wi-Fi on)
- watcher (10s): in charging mode, dc_det 1->0 (charger removed) -> poke latch ->
  Wi-Fi returns without reboot
- known limit: plugging a charger while the latch is armed needs a power-off to
  charge (the latch cannot be cleared live) — documented

## v25 (2026-11-07) — confirmed shared-net model + honest battery + wifi watchdog
**Hardware model confirmed by the user's own observations:**
- the power-port charge input and the OTG/hub 5V net are ONE net
- latch armed (wifi) -> own 5V on the net: PMIC reads "charger" (false icon)
  and cannot charge from itself
- latch off (device powered off, or boot with charger) -> charger feeds the
  net -> charging works (the user confirmed off-state charging)
- => charging while Wi-Fi is on is physically impossible on this board
**v25 logic:**
- boot: dc_det=1 AND hub down -> charging state (charger stays bound, latch not
  armed, Wi-Fi off)
- boot: battery -> arm latch -> wifi state; then unbind rk817-charger so the
  false "ac online" disappears (battery gauge is a separate driver)
- watchdog (10s): wifi state + hub down -> re-poke latch (fixes Wi-Fi not
  returning after an OTG-port charger event)
- watchdog: charging state + dc_det=0 (charger removed) -> arm latch -> wifi
**Known limitation:** plugging a charger while latched needs a power-off to
charge (latch cannot be cleared live).

## v26 (2026-11-07) — THE corrected design: stock topology + latch
**Why v24/v25 were wrong (found by re-reading our own v13 capture):**
- with the phy port enabled, the phy fires EXTCON_USB_VBUS_EN -> the charger
  sets otg_in=1 and **disables its plug/plug-out IRQs** ("disable_irq(
  charge->plugin_irq)") -> plugging a charger afterwards is never detected ->
  no charging. Stock never had this because its otg-port is disabled.
**v26 = stock topology + our latch:**
- u2phy_otg status=disabled (charger stays ignorant, plug detection alive)
- dwc2 dr_mode=host, phys=<&u2phy_otg> kept: lookup defers until the u2phy is
  up, then -ENODEV -> phyless probe with the clock running (v15 timing fix)
- dc_det restored (charging path)
- boot service arms the latch unconditionally, watchdog re-arms when the hub
  loses power (30 s)
- expectation: charging + Wi-Fi simultaneously, like stock

## v26 CONFIRMED ON HARDWARE (2026-11-07)
Live reading with the charger in the power port:
  ac: online=1 Charging | battery: Charging 3.899 V +869 mA | usbdevs=8 | wlan=2
=> charging and Wi-Fi run together, stock parity.
Boot-time detail: with a charger present the charger's 5 V powers the shared
net, so the hub enumerates before the latch is needed ("hub already up").
On battery the boot service arms the PMIC latch instead. The charger driver is
never told about OTG (otg-port disabled), so it keeps its plug detection and
charges normally in both cases.
The correct hardware model: the power-port input and the Wi-Fi hub 5 V net are
one net, and the rk817 can both charge from the input and feed the hub; the
earlier "mutually exclusive" conclusion was an artefact of the phy telling the
charger about OTG (which disables its plug IRQs).

## v27 (2026-11-06) — boot black screen FIXED + boot speed
**Root cause found (live boot analysis):**
- kernel draws the boot logo (drm_logo, freed at 11.4s) but NO vop modeset is
  visible in the whole boot console log; on stock RetroOZ the vop enables at
  1.4s. The Android handoff (SF/HWC taking DRM master) never enables the CRTC
  for the Android framebuffer. DisplayManagerService keeps the display in its
  initial "ON" state the whole boot, so nothing ever triggers a modeset.
  screencap showed the launcher fully composed -> compositor fine, panel scanout
  dead. First "Update mode to 480x854p60" only at 118s = the user's power-button
  press (sleep -> wake = DPMS off -> on = modeset -> picture appears).
**v27 fix (in the boot service, after sys.boot_completed):**
- wait for init.svc.bootanim to leave "running" (accepts empty for nobootanim)
- `input keyevent KEYCODE_POWER` twice, 1.5s apart -> DisplayPowerController
  does OFF->ON -> CRTC modeset -> panel shows the launcher. Verified live:
  automated cycle produced "Display device changed state OFF/ON" at 22:30:43/45
- alternatives tested and rejected: fb0/blank (no DPMS on this kernel),
  /sys/class/drm/card0-DSI-1/dpms (read-only)
**Boot speed (baseline 2026-11-06, charger in):**
- kernel->init 2.7s; zygote start 12.0s; preload 19.8->27.5s; PMS 31.4->35.7s;
  ams_ready 44.4s; enable_screen 52.2s; boot_completed ~57s; bootanim done 62.9s
- zram mkswap 731MB at ~14s (fstab zramsize=75%, fine)
- 191 packages; /data f2fs with lazytime+noatime (fine)
- applied: debug.sf.nobootanimation=1 in /vendor/build.prop (stock file has =0;
  /system is erofs, not writable) -> animation tail (~6s) gone + CPU/GPU freed
  during the heavy phase
- /system is erofs (read-only fs, no repack attempted yet)
- installed live on the card via adb (vendor remount rw); both verified in place

## v27 CONFIRMED ON HARDWARE (2026-11-06 22:38 boot)
- user: "its all alone i didnt touch it after the power on" — the panel comes
  up by itself; no button press needed anymore
- this boot vs pre-v27: enable_screen 52.2->47.7s; ready 62.9->51.1s (~12s);
  boot_completed ~48s; screen alive ~66s (auto)
- wake reason POWER_BUTTON from the injected keys; DisplayManager ON at
  22:37:24.4 right at the cycle; battery charged 38->44% with Wi-Fi up
- nobootanimation=1 active (bootanim exits instantly, ro.boottime.bootanim
  4.65s but no rendering)
- charging + wifi + hub all coexist (Charging, wlan0 inet, usbdevs=8)

## Performance phase — baseline + first wins (2026-11-06 22:45)
Baseline (idle, charging, launcher up):
- RAM: 975 MB total, 60 free / 369 available; zram 731 MB with ~29 MB in it
- CPU: 4x A35 schedutil 408-1296 MHz; GPU Mali-G31 simple_ondemand
  available 400/480/520/600 MHz but POLICY-CAPPED at 520 (min=max=520)
- temps: SoC 62C / GPU 61C (charging + idle), battery 18.8C
- top RSS: system_server 173MB, daijishou 172MB RSS/61MB PSS, systemui 159/57,
  + four cached andr36oid helper apps ~78MB RSS each
- 51 init services running; 191 packages; 23 third-party apps
Applied:
- am kill-all: MemAvailable 369 -> 435 MB (+66)
- device_config activity_manager max_cached_processes 12 (default 32)
- vendor thermal-mock HAL disabled (rc renamed .disabled; fake data anyway;
  real thermal is kernel soc/gpu zones) — takes effect next boot
- GPU max_freq unlocked to 600MHz (stock-firmware proven clock; cur stays 520
  idle, ramps under load) — LIVE-TEST PENDING in a game
- VLC PlaybackService force-stopped (was holding 18MB PSS in background)
RAM after: MemAvailable 442 MB.
Pending user decision: Thorium browser 341MB / VLC 137MB / Syncthing 116MB
on /data — removal is user's call (ROM sync via syncthing? browsing?).

## v28 (2026-11-06) — background janitor + runtime perf tuning
Service additions (all individually reversible):
- runtime tuning re-applied each boot: vm.swappiness=100 (fallback 60),
  mmc read_ahead_kb=512 (fallback 128), iostats=0 (fallback 1),
  GPU max_freq 600MHz (fallback 520000000), UI animation scales 0.5 (fallback 1.0)
- background janitor every 40s: am kill-all (cached only, safe) + force-stop
  thorium/vlc ONLY when not the foreground activity; syncthing deliberately
  excluded (needs to run to sync)
- game mode scripts in /data/local/tmp: gamemode (CPU floor 1.008GHz) /
  gamemode-off (408MHz) — tested both directions live
Zram: comp_algorithm shows "lzo lzo-rle [lz4]" -> already lz4 (brackets =
active), no change needed. swappiness raise is the right lever for 1GB+zram.
v28 installed live via vendor remount; janitor applies from next boot.

## v29 (2026-11-06) — L3/R3 fixed (mouse mode combo)
**Problem:** andr36oid's L3+R3 mouse mode never launched. Live getevent showed
the stick pushes emitting 0x2c4/0x2c5 (BTN_TRIGGER_HAPPY5/6) — our port's
mapping of sw11/sw12. andr36oid (and the R36S base) expects real L3/R3: the
R36S dtsi sends them as KEY_BACK/KEY_LEFTMETA (158/125) with a keylayout
mapping them to THUMBL/THUMBR; this image has no such .kl (Generic.kl applies,
which maps 317/318 -> BUTTON_THUMBL/THUMBR directly).
**Fix:** sw11 -> 0x13d (BTN_THUMBL 317), sw12 -> 0x13e (BTN_THUMBR 318).
No keylayout file needed; emulators see standard L3/R3 too.
**Verified:** rebuilt DTB, installed LIVE into BOOT via root (mount vfat p1),
rebooted; getevent now shows 013d/013e on stick pushes; boot clean (38s,
panel cycle ok). Backup of the previous DTB at /data/local/tmp/dtb-backup.
Note: /mnt is tmpfs — mountpoints created in one boot do not exist after
reboot (this is why the post-reboot cmp printed a false "mismatch").

## v29 CONFIRMED — mouse mode works (2026-11-06 23:23)
Live capture with the user holding the chords:
  joyMouse(586): "mouse mode on (toggle chord)"
  EventHub: New device event4 name='joyMouse' (virtual pointer, sources=0x2002)
  ModeToastReceiver ran (toast shown), then mode off ~22s later (second chord)
The joyMouse engine (/system/bin/joyMouse, init service, runs as root since
boot) reads /dev/input directly watching BUTTON_THUMBL/BUTTON_THUMBR evdev
codes — our v29 kernel codes (317/318) are exactly what it wants. No keylayout
file needed. Engine details: grabs the pad only while mouse mode is on,
reads /proc/device-tree/odroidgo3-joypad/andr36oid,missing-buttons (ours = "fn",
correct: this board lacks only the FN button), config via persist.sys.joymouse.*
props (none set = defaults), chord must be held ~2s, "toggle chord ignored:
a stick is being pushed" means sticks must be centered when pressing.
L3+R3 and SELECT+START both pressed during the window — both appear valid
chords (settings UI offers picker).

## v30 + v31 (2026-11-07) — right stick remapped, mouse mode PERFECT
**v30:** the right stick was axis-SWAPPED: horizontal pushes fired ABS_RY, ABS_RX
never moved. Root cause: this board's mux wiring differs from the R36S
(mux ch0 = right-stick X, ch1 = right-stick Y; the R36S is the reverse), and the
DT props `abs_rx_channel`/`abs_ry_channel` copied from the stock DTB are read by
NOBODY in the andr36oid driver — it only reads `amux-channel-mapping` (axis
order RY,RX,Y,X), defaulting to identity.
Fix: `amux-channel-mapping = <1 0 2 3>;` (ABS_RY->ch1, ABS_RX->ch0, left stick
unchanged). Verified: horizontal pushes now fire ABS_RX, vertical fire ABS_RY.
**v31:** after the remap the user measured a pure 180° inversion (right->left,
up->down). Polarity fix: `invert-absrx` back on (ch0 raw = -PX), NO
`invert-absry` (ch1 raw = +PY) — this matches the stock DTB exactly (it had
invert-absrx, no invert-absry; its abs_*_channel values + RetroOZ's driver were
self-consistent all along; the props were just invisible to our kernel).
**Result (user confirmed "perfect"):** right stick correct everywhere — mouse
mode cursor AND emulators' camera control (which was silently swapped/inverted
since the port began).
**Mouse mode final config:** toggle = Select+Start chord (hold ~1s);
pointer_stick = right stick (`persist.sys.joymouse.pointer_stick right`);
invert_ry=false, pointer_invert_x=false (all the compensation happens in the
kernel now). Props survive reboot; editable live or via JoyMouseSettings app.
**Engine facts (from source, andr36oid/android_device_gameconsole_common
joyMouse/):** native root daemon started by init at boot; scans /dev/input and
EVIOCGRABs the pad ONLY while mouse mode is on; virtual uinput mouse
(REL_X/Y + buttons); toggle chord default = Select+Start; pointer stick
defaults to RIGHT when the pad has one; productDevice=="r36s" skips the
right-Y flip other handhelds need (our board: kernel-side polarity fix instead
so the flip stays off); "toggle chord ignored: a stick is being pushed" = chord
needs centered sticks; config = persist.sys.joymouse.* read live (PropertyWatcher
waits on the property serial).

## v32 (2026-11-07) — Tailscale + remote play stack
- tailscaled (anasfanani/tailscale-android-cli v1.98.8-android, aarch64 static,
  6.3MB) at /data/local/tmp/ts/, CLI via symlink multicall
- /dev/tun present and usable by root -> full VPN mode --tun=tun0
- console joined the user's tailnet: the console 100.x.x.x
- ping to the PC 100.x.x.x: 0% loss (relayed ~130ms across networks;
  direct connects are LAN-fast when both ends share a network)
- service v32: tailscaled autostart (state-authenticated -> reconnects on boot)
  + janitor keep-alive; installed to /vendor live
- Moonlight v12.1 already preinstalled (com.limelight); stack = Moonlight +
  Sunshine (PC) + Tailscale. PyroWave reference noted (Adreno/LAN-only, later).

## v32 master rebuild kit (2026-11-07 01:40)
- test card found DYING (unstable reads: same sectors differ between reads;
  BOOT FAT corrupted: Panels/ I/O errors, top-level boot.ini/dtb lost)
- rescued from it: userdata-full.tar (1.5GB, all apps+data from p6) +
  essentials.tgz (tailscale auth state, pulled live earlier)
- fresh master built from the OFFICIAL release base (andr36oid-20261024-1750)
  via build-rgb10max2-image.sh (mtools-based BOOT rebuild after kernel-vfat
  write flakiness corrupted a previous attempt; mtools = deterministic)
- master: dist/andr36oid-rgb10max2-master.img md5 618f948d7055af948c362ff593433924
  contains: v31 DTB+boot.ini patched+profiles+vendor v32 service+tools+knob
- rebuild flow for the new card: flash master -> extend userdata to fill card
  (delete/recreate p6 + mkfs.f2fs + restore userdata-full.tar) -> push ts
  essentials -> boot -> connect wifi -> done (no re-login, apps intact)
