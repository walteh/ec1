package main

// package main

// import (
// 	"debug/elf"
// 	"fmt"
// 	"os"
// 	"runtime"
// 	"sort"
// 	"strings"

// 	"github.com/go-delve/delve/pkg/proc"
// )

// func debug() {
// 	// Use delve to decode the DWARF section
// 	binInfo := proc.NewBinaryInfo(runtime.GOOS, runtime.GOARCH)
// 	err := binInfo.AddImage(os.Args[1], 0)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	// Make a list of unique packages
// 	pkgs := make([]string, 0, len(binInfo.PackageMap))
// 	for _, fullPkgs := range binInfo.PackageMap {
// 		for _, fullPkg := range fullPkgs {
// 			exists := false
// 			for _, pkg := range pkgs {
// 				if fullPkg == pkg {
// 					exists = true
// 					break
// 				}
// 			}
// 			if !exists {
// 				pkgs = append(pkgs, fullPkg)
// 			}
// 		}
// 	}
// 	// Sort them for a nice output
// 	sort.Strings(pkgs)

// 	// Parse the ELF file ourselfs
// 	elfFile, err := elf.Open(os.Args[1])
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Get the symbol table
// 	symbols, err := elfFile.Symbols()
// 	if err != nil {
// 		panic(err)
// 	}

// 	usage := make(map[string]map[string]int)

// 	for _, sym := range symbols {
// 		if sym.Section == elf.SHN_UNDEF || sym.Section >= elf.SectionIndex(len(elfFile.Sections)) {
// 			continue
// 		}

// 		sectionName := elfFile.Sections[sym.Section].Name

// 		symPkg := ""
// 		for _, pkg := range pkgs {
// 			if strings.HasPrefix(sym.Name, pkg) {
// 				symPkg = pkg
// 				break
// 			}
// 		}
// 		// Symbol doesn't belong to a known package
// 		if symPkg == "" {
// 			continue
// 		}

// 		pkgStats := usage[symPkg]
// 		if pkgStats == nil {
// 			pkgStats = make(map[string]int)
// 		}

// 		pkgStats[sectionName] += int(sym.Size)
// 		usage[symPkg] = pkgStats
// 	}

// 	for _, pkg := range pkgs {
// 		sections, exists := usage[pkg]
// 		if !exists {
// 			continue
// 		}

// 		fmt.Printf("%s:\n", pkg)
// 		for section, size := range sections {
// 			fmt.Printf("%15s: %8d bytes\n", section, size)
// 		}
// 		fmt.Println()
// 	}
// }

// func debug1() {
// 	devicePath := "/dev/vport2p0" // Your specific device

// 	fmt.Printf("Debugging virtio console device: %s\n", devicePath)

// 	// 1. Check device file details
// 	checkDeviceFile(devicePath)

// 	// 2. Check sysfs information
// 	checkSysfsInfo("vport2p0")

// 	// 3. Check if device is ready
// 	checkDeviceReadiness(devicePath)

// 	// 4. Try different open modes
// 	tryOpenModes(devicePath)

// 	// 5. Check kernel logs for virtio console messages
// 	checkKernelLogs()
// }

// func checkDeviceFile(devicePath string) {
// 	fmt.Printf("\n=== Device File Information ===\n")

// 	stat, err := os.Stat(devicePath)
// 	if err != nil {
// 		fmt.Printf("Error stating device: %v\n", err)
// 		return
// 	}

// 	// Get device major/minor numbers
// 	sys := stat.Sys().(*syscall.Stat_t)
// 	major := (sys.Rdev >> 8) & 0xff
// 	minor := sys.Rdev & 0xff

// 	fmt.Printf("Device: %s\n", devicePath)
// 	fmt.Printf("Mode: %s\n", stat.Mode())
// 	fmt.Printf("Major: %d, Minor: %d\n", major, minor)
// 	fmt.Printf("Size: %d\n", stat.Size())
// 	fmt.Printf("ModTime: %s\n", stat.ModTime())
// }

// func checkSysfsInfo(portName string) {
// 	fmt.Printf("\n=== Sysfs Information ===\n")

// 	sysPath := "/sys/class/virtio-ports/" + portName

// 	// Check if sysfs entry exists
// 	if _, err := os.Stat(sysPath); err != nil {
// 		fmt.Printf("Sysfs path %s not found: %v\n", sysPath, err)
// 		return
// 	}

// 	// Read various sysfs attributes
// 	attributes := []string{"name", "dev", "active"}

// 	for _, attr := range attributes {
// 		attrPath := sysPath + "/" + attr
// 		if data, err := os.ReadFile(attrPath); err == nil {
// 			fmt.Printf("%s: %s", attr, strings.TrimSpace(string(data)))
// 		} else {
// 			fmt.Printf("%s: <not available>\n", attr)
// 		}
// 	}

// 	// Check if port is active
// 	activePath := sysPath + "/active"
// 	if data, err := os.ReadFile(activePath); err == nil {
// 		active := strings.TrimSpace(string(data))
// 		if active != "1" {
// 			fmt.Printf("WARNING: Port is not active (active=%s)\n", active)
// 		}
// 	}
// }

// func checkDeviceReadiness(devicePath string) {
// 	fmt.Printf("\n=== Device Readiness Check ===\n")

// 	// Try to get device status using ioctl or basic operations
// 	file, err := os.OpenFile(devicePath, os.O_RDONLY|syscall.O_NONBLOCK, 0)
// 	if err != nil {
// 		fmt.Printf("Cannot open device for reading: %v\n", err)
// 		return
// 	}
// 	defer file.Close()

// 	// Try a non-blocking read to see if device responds
// 	buffer := make([]byte, 1)
// 	_, err = file.Read(buffer)
// 	if err != nil {
// 		if err.Error() == "resource temporarily unavailable" {
// 			fmt.Printf("Device opened successfully (no data available)\n")
// 		} else {
// 			fmt.Printf("Device read error: %v\n", err)
// 		}
// 	} else {
// 		fmt.Printf("Device readable\n")
// 	}
// }

// func tryOpenModes(devicePath string) {
// 	fmt.Printf("\n=== Testing Different Open Modes ===\n")

// 	modes := []struct {
// 		name  string
// 		flags int
// 	}{
// 		{"O_RDONLY", os.O_RDONLY},
// 		{"O_WRONLY", os.O_WRONLY},
// 		{"O_RDWR", os.O_RDWR},
// 		{"O_WRONLY|O_NONBLOCK", os.O_WRONLY | syscall.O_NONBLOCK},
// 		{"O_RDWR|O_NONBLOCK", os.O_RDWR | syscall.O_NONBLOCK},
// 	}

// 	for _, mode := range modes {
// 		file, err := os.OpenFile(devicePath, mode.flags, 0)
// 		if err != nil {
// 			fmt.Printf("%-20s: FAILED - %v\n", mode.name, err)
// 		} else {
// 			fmt.Printf("%-20s: SUCCESS\n", mode.name)
// 			file.Close()
// 		}
// 	}
// }

// func checkKernelLogs() {
// 	fmt.Printf("\n=== Recent Kernel Messages ===\n")

// 	// Try to read kernel messages related to virtio console
// 	cmd := exec.Command("/bin/busybox", "dmesg", "-T")
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		fmt.Printf("Cannot read dmesg: %v\n", err)
// 		fmt.Println(string(output))
// 		return
// 	}

// 	lines := strings.Split(string(output), "\n")
// 	virtioLines := []string{}

// 	for _, line := range lines {
// 		if strings.Contains(strings.ToLower(line), "virtio") ||
// 			strings.Contains(strings.ToLower(line), "vport") ||
// 			strings.Contains(strings.ToLower(line), "console") {
// 			virtioLines = append(virtioLines, line)
// 		}
// 	}

// 	// Show last 10 relevant lines
// 	start := len(virtioLines) - 10
// 	if start < 0 {
// 		start = 0
// 	}

// 	for i := start; i < len(virtioLines); i++ {
// 		fmt.Println(virtioLines[i])
// 	}
// }

// // Additional helper function to wait for device to become ready
// func waitForDeviceReady(devicePath string, timeout time.Duration) error {
// 	fmt.Printf("\n=== Waiting for device to become ready ===\n")

// 	start := time.Now()
// 	for time.Since(start) < timeout {
// 		// Try to open the device
// 		file, err := os.OpenFile(devicePath, os.O_WRONLY|syscall.O_NONBLOCK, 0)
// 		if err == nil {
// 			file.Close()
// 			fmt.Printf("Device ready after %v\n", time.Since(start))
// 			return nil
// 		}

// 		fmt.Printf("Waiting... (%v)\n", err)
// 		time.Sleep(100 * time.Millisecond)
// 	}

// 	return fmt.Errorf("device not ready after %v", timeout)
// }

// func debug2() {
// 	fmt.Println("=== Checking Kernel Configuration for Virtio Console ===")

// 	// 1. Check current kernel version
// 	checkKernelVersion()

// 	// 2. Check kernel config if available
// 	checkKernelConfig()

// 	// 3. Check what's actually loaded/available
// 	checkVirtioStatus()

// 	// 4. Check PCI devices for virtio
// 	checkPCIDevices()

// 	// 5. Check driver binding status
// 	checkDriverBinding()
// }

// func checkKernelVersion() {
// 	fmt.Println("\n=== Kernel Version ===")

// 	cmd := exec.Command("uname", "-r")
// 	if output, err := cmd.Output(); err == nil {
// 		fmt.Printf("Kernel: %s\n", strings.TrimSpace(string(output)))
// 	}

// 	// Also check /proc/version for more details
// 	if data, err := os.ReadFile("/proc/version"); err == nil {
// 		fmt.Printf("Version: %s\n", strings.TrimSpace(string(data)))
// 	}
// }

// func checkKernelConfig() {
// 	fmt.Println("\n=== Kernel Configuration ===")

// 	// Common locations for kernel config
// 	configPaths := []string{
// 		"/proc/config.gz",
// 		"/boot/config-" + getKernelRelease(),
// 		"/boot/config",
// 		"/usr/src/linux/.config",
// 	}

// 	var configFound bool

// 	for _, path := range configPaths {
// 		if _, err := os.Stat(path); err == nil {
// 			fmt.Printf("Found config at: %s\n", path)
// 			checkVirtioConfigOptions(path)
// 			configFound = true
// 			break
// 		}
// 	}

// 	if !configFound {
// 		fmt.Println("Kernel config not found, checking alternative methods...")
// 		checkConfigFromModules()
// 	}
// }

// func getKernelRelease() string {
// 	cmd := exec.Command("uname", "-r")
// 	if output, err := cmd.Output(); err == nil {
// 		return strings.TrimSpace(string(output))
// 	}
// 	return ""
// }

// func checkVirtioConfigOptions(configPath string) {
// 	fmt.Printf("Checking virtio options in %s:\n", configPath)

// 	// Options we care about
// 	virtioOptions := []string{
// 		"CONFIG_VIRTIO",
// 		"CONFIG_VIRTIO_PCI",
// 		"CONFIG_VIRTIO_CONSOLE",
// 		"CONFIG_HVC_DRIVER",
// 		"CONFIG_VIRTIO_MMIO",
// 	}

// 	var cmd *exec.Cmd
// 	if strings.HasSuffix(configPath, ".gz") {
// 		cmd = exec.Command("zcat", configPath)
// 	} else {
// 		cmd = exec.Command("cat", configPath)
// 	}

// 	output, err := cmd.Output()
// 	if err != nil {
// 		fmt.Printf("Error reading config: %v\n", err)
// 		return
// 	}

// 	configLines := strings.Split(string(output), "\n")

// 	for _, option := range virtioOptions {
// 		found := false
// 		for _, line := range configLines {
// 			if strings.HasPrefix(line, option+"=") {
// 				fmt.Printf("  %s\n", line)
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			// Check if it's commented out (not set)
// 			commentedOption := "# " + option + " is not set"
// 			for _, line := range configLines {
// 				if strings.TrimSpace(line) == commentedOption {
// 					fmt.Printf("  %s\n", line)
// 					found = true
// 					break
// 				}
// 			}
// 		}
// 		if !found {
// 			fmt.Printf("  %s: <not found>\n", option)
// 		}
// 	}
// }

// func checkConfigFromModules() {
// 	fmt.Println("Checking from /sys/module (built-in modules):")

// 	builtinModules := []string{
// 		"virtio",
// 		"virtio_pci",
// 		"virtio_console",
// 		"hvc_console",
// 	}

// 	for _, module := range builtinModules {
// 		modulePath := "/sys/module/" + module
// 		if _, err := os.Stat(modulePath); err == nil {
// 			fmt.Printf("  %s: BUILT-IN (found in /sys/module)\n", module)
// 		} else {
// 			fmt.Printf("  %s: NOT AVAILABLE\n", module)
// 		}
// 	}
// }

// func checkVirtioStatus() {
// 	fmt.Println("\n=== Virtio Subsystem Status ===")

// 	// Check /sys/bus/virtio
// 	if entries, err := os.ReadDir("/sys/bus/virtio/devices"); err == nil {
// 		fmt.Printf("Virtio devices found: %d\n", len(entries))
// 		for _, entry := range entries {
// 			fmt.Printf("  Device: %s\n", entry.Name())
// 			checkVirtioDevice("/sys/bus/virtio/devices/" + entry.Name())
// 		}
// 	} else {
// 		fmt.Printf("No virtio bus found: %v\n", err)
// 	}

// 	// Check /sys/class/virtio-ports
// 	if entries, err := os.ReadDir("/sys/class/virtio-ports"); err == nil {
// 		fmt.Printf("Virtio ports found: %d\n", len(entries))
// 		for _, entry := range entries {
// 			fmt.Printf("  Port: %s\n", entry.Name())
// 		}
// 	} else {
// 		fmt.Printf("No virtio-ports class found: %v\n", err)
// 	}
// }

// func checkVirtioDevice(devicePath string) {
// 	// Check device ID
// 	if data, err := os.ReadFile(devicePath + "/device"); err == nil {
// 		deviceID := strings.TrimSpace(string(data))
// 		fmt.Printf("    Device ID: %s", deviceID)
// 		if deviceID == "0x0003" {
// 			fmt.Printf(" (virtio console)")
// 		}
// 		fmt.Printf("\n")
// 	}

// 	// Check if driver is bound
// 	if target, err := os.Readlink(devicePath + "/driver"); err == nil {
// 		fmt.Printf("    Driver: %s\n", strings.TrimPrefix(target, "../../../bus/virtio/drivers/"))
// 	} else {
// 		fmt.Printf("    Driver: <not bound>\n")
// 	}

// 	// Check status
// 	if data, err := os.ReadFile(devicePath + "/status"); err == nil {
// 		fmt.Printf("    Status: %s\n", strings.TrimSpace(string(data)))
// 	}
// }

// func checkPCIDevices() {
// 	fmt.Println("\n=== PCI Virtio Devices ===")

// 	cmd := exec.Command("lspci", "-nn")
// 	output, err := cmd.Output()
// 	if err != nil {
// 		fmt.Printf("lspci not available: %v\n", err)
// 		return
// 	}

// 	lines := strings.Split(string(output), "\n")
// 	for _, line := range lines {
// 		if strings.Contains(strings.ToLower(line), "virtio") {
// 			fmt.Printf("  %s\n", line)
// 		}
// 	}
// }

// func checkDriverBinding() {
// 	fmt.Println("\n=== Driver Binding Status ===")

// 	// Check available virtio drivers
// 	driversPath := "/sys/bus/virtio/drivers"
// 	if entries, err := os.ReadDir(driversPath); err == nil {
// 		fmt.Printf("Available virtio drivers:\n")
// 		for _, entry := range entries {
// 			fmt.Printf("  %s\n", entry.Name())

// 			// Check what devices are bound to this driver
// 			bindPath := driversPath + "/" + entry.Name()
// 			if bindEntries, err := os.ReadDir(bindPath); err == nil {
// 				for _, bindEntry := range bindEntries {
// 					if strings.HasPrefix(bindEntry.Name(), "virtio") {
// 						fmt.Printf("    -> bound to: %s\n", bindEntry.Name())
// 					}
// 				}
// 			}
// 		}
// 	} else {
// 		fmt.Printf("No virtio drivers directory found: %v\n", err)
// 	}
// }

// func debug3() {
// 	// Additional debugging: show the exact error from opening the device
// 	fmt.Println("\n=== Detailed Device Error Analysis ===")

// 	devicePath := "/dev/vport2p0"

// 	// Try to get more specific error information
// 	file, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
// 	if err != nil {
// 		fmt.Printf("Open error: %v\n", err)
// 		fmt.Printf("Error type: %T\n", err)

// 		// Check if it's a specific system error
// 		if pathErr, ok := err.(*os.PathError); ok {
// 			fmt.Printf("Path error: %v\n", pathErr.Err)
// 			fmt.Printf("Syscall: %s\n", pathErr.Op)
// 			fmt.Printf("Path: %s\n", pathErr.Path)
// 		}
// 	} else {
// 		file.Close()
// 		fmt.Printf("Device opened successfully!\n")
// 	}
// }

// func debug4(ctx context.Context) {
// 	slog.InfoContext(ctx, "=== Comprehensive Virtio Console Debug ===")

// 	// 1. Check all vport devices
// 	vportPaths := []string{
// 		"/dev/vport1p0", "/dev/vport2p0", "/dev/vport3p0", "/dev/vport4p0",
// 	}

// 	for _, vportPath := range vportPaths {
// 		slog.InfoContext(ctx, "checking vport device", "path", vportPath)

// 		// Check if file exists
// 		if _, err := os.Stat(vportPath); err != nil {
// 			slog.WarnContext(ctx, "vport device stat failed", "path", vportPath, "error", err)
// 			continue
// 		}

// 		// Try to open
// 		file, err := os.OpenFile(vportPath, os.O_WRONLY|syscall.O_NONBLOCK, 0)
// 		if err != nil {
// 			slog.WarnContext(ctx, "vport device open failed", "path", vportPath, "error", err)
// 		} else {
// 			slog.InfoContext(ctx, "vport device opened successfully", "path", vportPath)
// 			file.Close()
// 		}
// 	}

// 	// 2. Check sysfs virtio-ports
// 	portDir := "/sys/class/virtio-ports"
// 	if entries, err := os.ReadDir(portDir); err == nil {
// 		for _, entry := range entries {
// 			portName := entry.Name()
// 			slog.InfoContext(ctx, "checking sysfs port", "port", portName)

// 			// Check active status
// 			activePath := filepath.Join(portDir, portName, "active")
// 			if data, err := os.ReadFile(activePath); err == nil {
// 				active := strings.TrimSpace(string(data))
// 				slog.InfoContext(ctx, "port active status", "port", portName, "active", active)
// 			}

// 			// Check device numbers
// 			devPath := filepath.Join(portDir, portName, "dev")
// 			if data, err := os.ReadFile(devPath); err == nil {
// 				dev := strings.TrimSpace(string(data))
// 				slog.InfoContext(ctx, "port device number", "port", portName, "dev", dev)
// 			}
// 		}
// 	}

// 	// 3. Check virtio devices
// 	virtioDir := "/sys/bus/virtio/devices"
// 	if entries, err := os.ReadDir(virtioDir); err == nil {
// 		for _, entry := range entries {
// 			deviceName := entry.Name()
// 			devicePath := filepath.Join(virtioDir, deviceName)

// 			// Check device ID
// 			if data, err := os.ReadFile(filepath.Join(devicePath, "device")); err == nil {
// 				deviceID := strings.TrimSpace(string(data))
// 				if deviceID == "0x0003" { // virtio console
// 					slog.InfoContext(ctx, "found virtio console device",
// 						"device", deviceName,
// 						"id", deviceID)

// 					// Check driver binding
// 					if target, err := os.Readlink(filepath.Join(devicePath, "driver")); err == nil {
// 						driver := filepath.Base(target)
// 						slog.InfoContext(ctx, "virtio console driver",
// 							"device", deviceName,
// 							"driver", driver)
// 					}
// 				}
// 			}
// 		}
// 	}
// }
