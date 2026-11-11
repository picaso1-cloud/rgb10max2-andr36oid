//go:build linux

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func listDevices() ([]Device, error) {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil, err
	}
	skip := []string{"loop", "zram", "dm-", "md", "sr", "ram", "nbd", "fd"}
	rootDev := ""
	for _, line := range strings.Split(readTrim("/proc/mounts"), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[1] == "/" {
			rootDev = f[0]
		}
	}
	var out []Device
	for _, e := range entries {
		name := e.Name()
		bad := false
		for _, p := range skip {
			if strings.HasPrefix(name, p) {
				bad = true
			}
		}
		if bad {
			continue
		}
		base := filepath.Join("/sys/block", name)
		real, _ := filepath.EvalSymlinks(base)
		usb := strings.Contains(real, "/usb")
		removable := readTrim(base+"/removable") == "1"
		mmc := strings.HasPrefix(name, "mmcblk")
		if !removable && !usb && !mmc {
			continue
		}
		/* never offer a rotating USB hard drive as a flash target */
		rotational := readTrim(base+"/queue/rotational") == "1"
		if rotational && !removable {
			continue
		}
		sectors, _ := strconv.ParseUint(readTrim(base+"/size"), 10, 64)
		size := sectors * 512
		if size == 0 {
			continue
		}
		path := "/dev/" + name
		if rootDev != "" && strings.HasPrefix(rootDev, path) {
			continue
		}
		d := Device{
			Path:      path,
			Model:     readTrim(base + "/device/model"),
			Size:      size,
			Removable: removable,
			Bus:       "usb",
		}
		if d.Model == "" {
			d.Model = readTrim(base + "/device/name")
		}
		if mmc && !usb {
			d.Bus = "mmc"
		}
		out = append(out, d)
	}
	return out, nil
}

func mountedParts(dev string) []string {
	var parts []string
	data := readTrim("/proc/mounts")
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		if f[0] == dev || strings.HasPrefix(f[0], dev+"p") { // sdb1 or mmcblk0p1
			if strings.HasPrefix(f[0], dev) {
				parts = append(parts, f[0])
			}
		}
	}
	return parts
}

func writeImage(dev Device, image string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("writing needs root: run with sudo")
	}
	if parts := mountedParts(dev.Path); len(parts) > 0 {
		fmt.Printf("unmounting %v\n", parts)
		for _, p := range parts {
			if out, err := exec.Command("umount", p).CombinedOutput(); err != nil {
				return fmt.Errorf("could not unmount %s: %v (%s)", p, err, strings.TrimSpace(string(out)))
			}
		}
	}

	src, err := os.Open(image)
	if err != nil {
		return err
	}
	defer src.Close()
	info, _ := src.Stat()

	dst, err := os.OpenFile(dev.Path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", dev.Path, err)
	}
	defer dst.Close()

	fmt.Printf("writing %s to %s ...\n", human(uint64(info.Size())), dev.Path)
	if err := copyWithProgress(dst, src, info.Size()); err != nil {
		return err
	}
	return dst.Sync()
}

func verifyImage(dev Device, size int64, want [32]byte) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("verification needs root: run with sudo")
	}
	f, err := os.Open(dev.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.CopyN(h, f, size); err != nil {
		return err
	}
	var got [32]byte
	copy(got[:], h.Sum(nil))
	if got != want {
		return fmt.Errorf("read-back mismatch — the card is bad or the write failed")
	}
	return nil
}
