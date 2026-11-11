# 09 — Remote play (Steam games from anywhere)

The console streams PC games via **Moonlight + Sunshine over Tailscale**. The
whole stack is installed and working; this page documents every piece so it can
be rebuilt from scratch.

## The stack

| Layer | Component | Why this one |
|---|---|---|
| Android client | **Moonlight** (`com.limelight`, v12.1, preinstalled) | Lightest client: ~20 MB RAM, hardware H.264/HEVC decode via MediaCodec (Mali-G31 fine), the console's joypad is a native Android gamepad |
| PC host | **Sunshine** (open-source; the Vibepollo fork adds features) | Streams any game/app — Steam included — pairs with Moonlight, works on every GPU |
| Network | **Tailscale** (CLI build, no app) | Console joins your tailnet → Moonlight talks to the PC's `100.x.y.z` address from anywhere; WireGuard crypto, no port forwarding |
| Rejected | Steam Link (official) | Heavier on 1 GB RAM, no advantage here |
| Rejected (for now) | PyroWave/Artemis forks ([pyrowave-streaming](https://github.com/joemossjr16/pyrowave-streaming)) | Experimental intra-only wavelet codec; targets **Adreno** GPUs and needs LAN-only hundreds of Mbps — revisit only for LAN streaming on different hardware |

## Console side (installed and working)

- Binary: `tailscaled` from [anasfanani/tailscale-android-cli](https://github.com/anasfanani/tailscale-android-cli)
  releases (v1.98.8-android, aarch64, 6.3 MB, static, Android 21+). One binary:
  symlink it as `tailscale` for the CLI (multicall dispatch on argv[0]).
- Location: `/data/local/tmp/ts/` (binary, `tailscale` symlink, `state/`,
  `tsd.log`, unix socket `tailscaled.sock`).
- Started by the boot service (v32) when `state/tailscaled.state` exists —
  authentication is stored, so plain `tailscaled` reconnects on every boot.
  The janitor loop restarts it if it ever dies.
- The kernel has working TUN (`/dev/tun`, root can create tuntap devices) —
  full VPN mode, `--tun=tun0`, console gets a real tailnet IP.
- Console tailnet IP at setup: `100.x.x.x` (`the console`).
- Joining a NEW install: run the daemon, then `tailscale --socket=... up` and
  open the printed `https://login.tailscale.com/a/...` URL on any device to
  authenticate. RAM cost ~40-60 MB; the janitor's kill-all never touches it
  (native daemon, not an Android app process).

## PC side

1. Install **Sunshine** ([github.com/LizardByte/Sunshine](https://github.com/LizardByte/Sunshine)) on the gaming PC.
2. Log into Sunshine's web UI (`https://localhost:47990`), set credentials.
3. **Also install Tailscale on the PC**, log into the same tailnet.
4. In Moonlight on the console: **Add Host manually** → the PC's tailnet IP
   (`100.x.x.x` for `the PC` at setup time) → pair with the PIN
   Sunshine shows.
5. Steam games: add Steam (Big Picture or a specific game) as an app in
   Sunshine, or just stream the desktop and launch Steam.

## Latency reality

- **Same network (console + PC at home):** Tailscale connects **directly**,
  ~1-3 ms — streaming feels native.
- **Across networks/internet:** quality depends on path. If a **direct**
  WireGuard connection forms (most home NATs allow it), latency = the internet
  path, playable. If it falls back to a **DERP relay**, latency jumps
  (measured 80-250 ms console→home devices while relaying). Check with
  `tailscale status` — `direct` vs `relay "xxx"`.
- Improving direct-ness: enable UPnP on the router, or run a tailnet node as
  exit node / subnet router near the PC, or self-host a DERP close to both.

## Console quick-commands

```sh
TS=/data/local/tmp/ts/tailscale; SK=--socket=/data/local/tmp/ts/tailscaled.sock
$TS $SK status            # tailnet overview, direct/relay per peer
$TS $SK ip -4             # console tailnet IP
ping -c4 100.x.x.x    # reach the PC
pidof tailscaled          # daemon alive?
```

(Over adb, or through the root-command daemon `/data/local/tmp/cmd.in`.)
