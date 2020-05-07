package prepare

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/golang/glog"
)

const (
	SYS_BUS_PCI_DEVICES = SYS + "bus/pci/devices/"
	SYS_BUS_PCI_PROBE   = SYS + "bus/pci/drivers_probe"
)

// /sys/bus/pci/devices/0000:06:00.0/driver_override - vfio-pci or (null)
// /sys/bus/pci/devices/0000:06:00.0/ - default bus type is "pci"
// "realpath /sys/bus/pci/devices/0000:06:00.0/" => /sys/devices/pci0000:00/0000:00:03.2/0000:06:00.0
// unbind => echo "0000:06:00.0" > /sys/devices/pci0000:00/0000:00:03.2/0000:06:00.0/driver/unbind
// override support => /sys/devices/pci0000:00/0000:00:03.2/0000:06:00.0/driver_override present
// bind driver => echo "vfio-pci" >/sys/devices/pci0000:00/0000:00:03.2/0000:06:00.0/driver_override
// probe after bind => echo "0000:06:00.0" > "/sys/bus/pci/drivers_probe"
// verify driver bind => /sys/devices/pci0000:00/0000:00:03.2/0000:06:00.0/driver should be valid symbolic link
// driver modprobe ?

func BindDriver(pciAddress, driver string) error {
	if driver == "" {
		clearDriverOverride(pciAddress)
		return nil
	}
	curDriver := getInterfaceDriver(pciAddress)
	if curDriver == driver {
		glog.Infof("Device %s is already bound to driver %s", pciAddress, driver)
		return nil
	} else if curDriver != "" {
		err := unBindDriver(pciAddress)
		if err != nil {
			glog.Errorf("BindDriver: Failed to unbind driver for pci %s: %v", pciAddress, err)
			return err
		}
	} else {
		// If the driver is unbound, ensure the override is cleared
		clearDriverOverride(pciAddress)

		// Driver is invalid, check if the device is valid
		pciPath := path.Join(SYS_BUS_PCI_DEVICES, pciAddress)
		if !isSymLink(pciPath) {
			err := fmt.Errorf("BindDriver: Invalid pci pci %s", pciPath)
			return err
		}
	}

	// Bind the given driver
	realPath := getInterfaceRealPath(pciAddress)
	overridePath := path.Join(realPath, "driver_override")
	err := writeFile(overridePath, driver)
	if err != nil {
		return err
	}
	glog.Infof("BindDriver: Driver %s is bound to device %s, verifying now...", driver, pciAddress)

	probeDriver(pciAddress)

	curDriver = getInterfaceDriver(pciAddress)
	if curDriver != driver {
		glog.Errorf("BindDriver: Driver bind verification failed, current bound driver is (%s)", curDriver)
		glog.Errorf("BindDriver: Reverting the driver override, to load the default driver")
		clearDriverOverride(pciAddress)
		probeDriver(pciAddress)
	}
	return nil
}

func probeDriver(pciAddress string) {
	err := writeFile(SYS_BUS_PCI_PROBE, pciAddress)
	if err != nil {
		glog.Errorf("probeDriver: Driver probe failed for device %s: %v", pciAddress, err)
	}
}

func clearDriverOverride(pciAddress string) {
	realPath := getInterfaceRealPath(pciAddress)
	overridePath := path.Join(realPath, "driver_override")
	cmd := exec.Command("sh", "-c", "echo > "+overridePath)
	_, err := cmd.Output()
	if err != nil {
		glog.Errorf("clearDriverOverride: Command err: %v", err)
	}

}

func writeFile(filePath, fileData string) error {
	f, err := os.OpenFile(filePath, os.O_WRONLY, 0)
	if err != nil {
		glog.Errorf("writeFile: Failed to open file %s: %v", filePath, err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fileData)
	if err != nil {
		glog.Errorf("writeFile: Failed to write data %s to file %s: %v", fileData, filePath, err)
		return err
	}
	return nil
}

func unBindDriver(pciAddress string) error {
	clearDriverOverride(pciAddress)
	realPath := getInterfaceRealPath(pciAddress)
	unbindPath := path.Join(realPath, "driver", "unbind")
	_, err := os.Stat(unbindPath)
	if os.IsNotExist(err) {
		glog.Errorf("KRS; unBindDriver: Failed to write to unbind file: %v", err)
		// Driver not bound to the pci address
		return nil
	}

	err = writeFile(unbindPath, pciAddress)
	if err != nil {
		glog.Errorf("unBindDriver: Failed to write to unbind file: %v", err)
		return err
	}
	return nil
}

func getInterfaceRealPath(pciAddress string) string {
	pciPath := path.Join(SYS_BUS_PCI_DEVICES, pciAddress)
	if !isSymLink(pciPath) {
		glog.Errorf("getInterfaceRealPath: %s is not a symbolic link", pciPath)
		return ""
	}

	pciLink, err := os.Readlink(pciPath)
	if err != nil {
		glog.Errorf("getInterfaceRealPath: Failed to readlink for path %s: %v", pciPath, err)
		return ""
	}

	realPath := path.Join(SYS_BUS_PCI_DEVICES, pciLink)
	return realPath
}

func getInterfaceDriver(pciAddress string) string {
	realPath := getInterfaceRealPath(pciAddress)
	driverPath := path.Join(realPath, "driver")
	driverLink, err := os.Readlink(driverPath)
	if err != nil {
		glog.Errorf("getInterfaceDriver: Failed to readlink for path %s: %v", driverPath, err)
		return ""
	}

	_, f := path.Split(driverLink)
	if f == "" {
		glog.Errorf("getInterfaceDriver: Unable to get driver from %s", driverLink)
		return ""
	}

	return f
}

func isSymLink(path string) bool {
	stat, err := os.Lstat(path)
	if err != nil {
		glog.Errorf("isSymLink: Failed to Lstat of path (%s): %v", path, err)
		return false
	}

	mode := stat.Mode()
	if mode&os.ModeSymlink != 0 {
		return true
	}
	return false
}
