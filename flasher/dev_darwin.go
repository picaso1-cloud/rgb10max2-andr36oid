//go:build darwin

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func listDevices() ([]Device, error) {
	out, err := exec.Command("diskutil", "list", "external").Output()
	if err != nil {
		return nil, fmt.Errorf("diskutil: %w", err)
	}
	var devs []Device
	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "/dev/disk") {
			continue
		}
		path := strings.Fields(line)[0]
		if seen[path] {
			continue
		}
		seen[path] = true
		info, _ := exec.Command("diskutil", "info", path).Output()
		removable := false
		model := ""
		var size uint64
		for _, l := range strings.Split(string(info), "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Device / Media Name:") {
				model = strings.TrimSpace(strings.TrimPrefix(l, "Device / Media Name:"))
			}
			if strings.HasPrefix(l, "Removable Media:") &&
				strings.Contains(l, "Removable") {
				removable = true
			}
			if strings.HasPrefix(l, "Disk Size:") {
				// Disk Size: 31.9 GB (31914983424 Bytes) (exactly ...)
				if i := strings.Index(l, "("); i >= 0 {
					f := strings.Fields(l[i+1:])
					if len(f) > 0 {
						var b uint64
						fmt.Sscanf(f[0], "%d", &b)
						size = b
					}
				}
			}
		}
		if !removable {
			continue
		}
		devs = append(devs, Device{Path: path, Model: model, Size: size, Removable: true, Bus: "usb"})
	}
	return devs, nil
}

func writeImage(dev Device, image string) error {
	if err := exec.Command("diskutil", "unmountDisk", "force", dev.Path).Run(); err != nil {
		return fmt.Errorf("could not unmount %s: %w", dev.Path, err)
	}
	raw := strings.Replace(dev.Path, "/dev/disk", "/dev/rdisk", 1)

	src, err := os.Open(image)
	if err != nil {
		return err
	}
	defer src.Close()
	info, _ := src.Stat()

	dst, err := os.OpenFile(raw, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open %s (try sudo): %w", raw, err)
	}
	defer dst.Close()

	fmt.Printf("writing %s to %s ...\n", human(uint64(info.Size())), raw)
	if err := copyWithProgress(dst, src, info.Size()); err != nil {
		return err
	}
	return dst.Sync()
}

func verifyImage(dev Device, size int64, want [32]byte) error {
	raw := strings.Replace(dev.Path, "/dev/disk", "/dev/rdisk", 1)
	f, err := os.Open(raw)
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
