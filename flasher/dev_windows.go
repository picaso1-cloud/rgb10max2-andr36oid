//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Windows support is experimental: it uses diskpart to clean the disk, then
// writes to \\.\PhysicalDriveN. Run from an elevated (Administrator) console.

type psDisk struct {
	Number       int
	FriendlyName string
	Size         int64
	SerialNumber string
}

func listDevices() ([]Device, error) {
	script := `Get-Disk | Where-Object { $_.BusType -eq 'USB' -or $_.BusType -eq 'SD' } | ` +
		`Select-Object Number,FriendlyName,Size,SerialNumber | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return nil, fmt.Errorf("powershell: %w", err)
	}
	var disks []psDisk
	if strings.TrimSpace(string(out)) == "" {
		return nil, nil
	}
	if err := json.Unmarshal(out, &disks); err != nil {
		// a single object is returned without an array
		var one psDisk
		if e2 := json.Unmarshal(out, &one); e2 == nil {
			disks = []psDisk{one}
		} else {
			return nil, err
		}
	}
	var devs []Device
	for _, d := range disks {
		devs = append(devs, Device{
			Path:      fmt.Sprintf(`\\.\PhysicalDrive%d`, d.Number),
			Model:     d.FriendlyName,
			Size:      uint64(d.Size),
			Removable: true,
			Bus:       "usb",
		})
	}
	return devs, nil
}

func diskNumber(path string) (int, bool) {
	i := strings.LastIndex(path, "PhysicalDrive")
	if i < 0 {
		return 0, false
	}
	n, err := strconv.Atoi(path[i+len("PhysicalDrive"):])
	return n, err == nil
}

func writeImage(dev Device, image string) error {
	num, ok := diskNumber(dev.Path)
	if !ok {
		return fmt.Errorf("unexpected device path %q", dev.Path)
	}

	dir, err := os.MkdirTemp("", "rgb10max2-flasher")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	script := filepath.Join(dir, "clean.txt")
	body := fmt.Sprintf("select disk %d\r\nclean\r\nexit\r\n", num)
	if err := os.WriteFile(script, []byte(body), 0o600); err != nil {
		return err
	}
	fmt.Printf("cleaning disk %d with diskpart...\n", num)
	if out, err := exec.Command("diskpart", "/s", script).CombinedOutput(); err != nil {
		return fmt.Errorf("diskpart failed (run as Administrator): %v\n%s", err, out)
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
