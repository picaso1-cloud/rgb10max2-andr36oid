# rgb10max2-flasher

Cross-platform SD writer for the RGB10 Max 2 andr36oid image, with **SHA-256
read-back verification** so a corrupting card is detected before you boot it.

## Build

Requires Go 1.21+.

```sh
./build.sh
```

Produces:

```
dist/rgb10max2-flasher-linux-amd64
dist/rgb10max2-flasher-linux-arm64
dist/rgb10max2-flasher-darwin-amd64
dist/rgb10max2-flasher-darwin-arm64
dist/rgb10max2-flasher-windows-amd64.exe
```

## Use

```sh
sudo ./dist/rgb10max2-flasher-linux-amd64 list
sudo ./dist/rgb10max2-flasher-linux-amd64 flash ../RGB10MAX2-andr36oid.img
```

It shows the removable cards, asks you to pick one and type `YES`, writes the
image, then reads it back and compares SHA-256.

Options:

| Option | Meaning |
|---|---|
| `--device /dev/sdX` | skip the picker |
| `--yes` | skip the confirmation |
| `--no-verify` | skip read-back (not recommended) |

If `IMAGE.sha256` exists next to the image, it is checked **before** writing.

## Platform notes

- **Linux**: run as root; mounted partitions of the target are unmounted
  automatically. The system disk is never listed.
- **macOS**: `diskutil unmountDisk` is used; writes to `/dev/rdisk*`.
- **Windows**: *experimental*. Run as Administrator. It uses `diskpart` to clean
  the selected disk and then writes to `\\.\PhysicalDriveN`, followed by the same
  read-back verification. Until this has more testing, the official andr36oid
  flasher (manual image) is the safer Windows path.

## Why read-back verification

Cheap and counterfeit cards often accept writes and then return different data.
This tool reads the image back and compares hashes; a mismatch means the card
must be replaced. It is the same idea the official andr36oid flasher uses.
