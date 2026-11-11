// rgb10max2-flasher — write an andr36oid RGB10 Max 2 image to an SD card
// with optional SHA-256 read-back verification.
//
// Usage:
//   rgb10max2-flasher list
//   rgb10max2-flasher flash <image.img> [--device /dev/sdX] [--yes] [--no-verify]
//   rgb10max2-flasher verify <image.img> [--device /dev/sdX] [--yes]
//   rgb10max2-flasher version
//
// Linux and macOS are supported natively. Windows support is experimental:
// it uses diskpart to clean the disk and then writes to \\.\PhysicalDriveN;
// test it on a spare card first.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const version = "0.1.0"

type Device struct {
	Path      string
	Model     string
	Size      uint64
	Removable bool
	Bus       string
}

func human(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("rgb10max2-flasher %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	case "list":
		devs, err := listDevices()
		if err != nil {
			die(err)
		}
		printDevices(devs)
	case "flash":
		runFlash(os.Args[2:], true)
	case "verify":
		runFlash(os.Args[2:], false)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`rgb10max2-flasher — SD card writer for the unofficial andr36oid
RGB10 Max 2 port.

  list                          show removable cards/disks
  flash  <image> [options]      write the image
  verify <image> [options]      only read back and compare
  version

options:
  --device <path>   target device (skip the interactive picker)
  --yes             do not ask for confirmation
  --no-verify       skip the read-back check (not recommended)

Examples:
  sudo ./rgb10max2-flasher list
  sudo ./rgb10max2-flasher flash RGB10MAX2-andr36oid.img
`)
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func printDevices(devs []Device) {
	if len(devs) == 0 {
		fmt.Println("no removable devices found (insert the SD card and try again)")
		return
	}
	fmt.Printf("%-3s %-14s %9s  %-8s %-6s %s\n", "#", "device", "size", "bus", "rem", "model")
	for i, d := range devs {
		rem := "no"
		if d.Removable {
			rem = "yes"
		}
		fmt.Printf("%-3d %-14s %9s  %-8s %-6s %s\n", i+1, d.Path, human(d.Size), d.Bus, rem, d.Model)
	}
}

// ---------------- image helpers ----------------

func fileSHA256(path string) ([32]byte, int64, error) {
	var zero [32]byte
	f, err := os.Open(path)
	if err != nil {
		return zero, 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return zero, n, err
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum, n, nil
}

// checkSidecar verifies <image>.sha256 when present.
func checkSidecar(image string) error {
	data, err := os.ReadFile(image + ".sha256")
	if err != nil {
		return nil // optional
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return fmt.Errorf("%s.sha256 is empty", image)
	}
	want := strings.ToLower(fields[0])
	got, _, err := fileSHA256(image)
	if err != nil {
		return err
	}
	if hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("image SHA-256 mismatch:\n  want %s\n  got  %s", want, hex.EncodeToString(got[:]))
	}
	fmt.Println("image SHA-256: OK")
	return nil
}

func runFlash(args []string, doWrite bool) {
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	image := args[0]
	var devPath string
	yes := false
	verify := true
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--device":
			if i+1 >= len(args) {
				die(errors.New("--device needs a path"))
			}
			devPath = args[i+1]
			i++
		case "--yes", "-y":
			yes = true
		case "--no-verify":
			verify = false
		default:
			die(fmt.Errorf("unknown option %q", args[i]))
		}
	}

	if _, err := os.Stat(image); err != nil {
		die(err)
	}
	if err := checkSidecar(image); err != nil {
		die(err)
	}

	devs, err := listDevices()
	if err != nil {
		die(err)
	}
	if len(devs) == 0 {
		die(errors.New("no removable devices found"))
	}

	var dev Device
	if devPath != "" {
		for _, d := range devs {
			if d.Path == devPath {
				dev = d
				break
			}
		}
		if dev.Path == "" {
			die(fmt.Errorf("%s is not in the removable list (run 'list')", devPath))
		}
	} else {
		printDevices(devs)
		fmt.Print("\nselect the card number: ")
		var pick int
		if _, err := fmt.Scanln(&pick); err != nil || pick < 1 || pick > len(devs) {
			die(errors.New("invalid selection"))
		}
		dev = devs[pick-1]
	}

	info, err := os.Stat(image)
	if err != nil {
		die(err)
	}
	fmt.Printf("\nimage : %s (%s)\n", image, human(uint64(info.Size())))
	fmt.Printf("card  : %s  %s  %s\n", dev.Path, human(dev.Size), dev.Model)
	if uint64(info.Size()) > dev.Size {
		die(fmt.Errorf("image (%s) does not fit on %s (%s)", human(uint64(info.Size())), dev.Path, human(dev.Size)))
	}
	if devPath == "" || !yes {
		fmt.Printf("\nALL DATA ON %s WILL BE DESTROYED.\nType YES to continue: ", dev.Path)
		var answer string
		fmt.Scanln(&answer)
		if answer != "YES" {
			fmt.Println("aborted")
			return
		}
	}

	if doWrite {
		if err := writeImage(dev, image); err != nil {
			die(err)
		}
		fmt.Println("\nwrite: done")
	}
	if verify {
		want, size, err := fileSHA256(image)
		if err != nil {
			die(err)
		}
		fmt.Println("verifying (reading the card back)...")
		if err := verifyImage(dev, size, want); err != nil {
			die(err)
		}
		fmt.Println("verify: OK — the card matches the image")
	}
}

// ---------------- shared progress writer ----------------

func copyWithProgress(dst io.Writer, src io.Reader, total int64) error {
	const chunk = 4 << 20
	buf := make([]byte, chunk)
	var done int64
	start := time.Now()
	last := time.Now()
	for {
		n, rerr := io.ReadFull(src, buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if time.Since(last) > 700*time.Millisecond {
				el := time.Since(start).Seconds()
				rate := float64(done) / 1024 / 1024 / el
				pct := float64(done) * 100 / float64(total)
				eta := 0.0
				if rate > 0 {
					eta = float64(total-done) / 1024 / 1024 / rate
				}
				fmt.Printf("\r  %5.1f%%  %s / %s  %.1f MB/s  ETA %02.0f:%02.0f   ",
					pct, human(uint64(done)), human(uint64(total)), rate,
					eta/60, float64(int(eta)%60))
				last = time.Now()
			}
		}
		if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	fmt.Printf("\r  100.0%%  %s written                    \n", human(uint64(done)))
	return nil
}

func parseFlagInt(s string) int64 { v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64); return v }

func have(cmd string) bool { _, err := exec.LookPath(cmd); return err == nil }
