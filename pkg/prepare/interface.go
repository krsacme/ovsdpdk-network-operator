package prepare

import (
	"fmt"
	"os"
	"path"

	"github.com/golang/glog"

	ovsdpdkv1 "github.com/krsacme/ovsdpdk-network-operator/pkg/apis/ovsdpdknetwork/v1"
)

const (
	SYS           = "/host/sys/"
	SYS_CLASS_NET = SYS + "class/net/"
)

func PrepareOvsBridgeConfig(ifaceConfigs *[]ovsdpdkv1.InterfaceConfig) error {
	for _, ifaceConfig := range *ifaceConfigs {
		if len(ifaceConfig.NicSelector.Devices) == 0 && len(ifaceConfig.NicSelector.IfNames) == 0 {
			err := fmt.Errorf("Devices or IfNames must be specified")
			glog.Errorf("PrepareOvsBridgeConfig: %v", err)
			return err
		}

		if len(ifaceConfig.NicSelector.Devices) > 0 && len(ifaceConfig.NicSelector.IfNames) > 0 {
			err := fmt.Errorf("Both Devices and IfNames cannot be specified")
			glog.Errorf("PrepareOvsBridgeConfig: %v", err)
			return err
		}

		pciAddressList, err := GetPciAddressList(ifaceConfig.NicSelector)
		if err != nil {
			glog.Errorf("PrepareOvsBridgeConfig: Failed to get PCI address: %v", err)
			return err
		}

		err = bindDriver(ifaceConfig.Driver, pciAddressList)
		if err != nil {
			glog.Errorf("PrepareOvsBridgeConfig: Failed to bind driver")
			return err
		}

		err = addBridge(ifaceConfig.Bridge)
		if err != nil {
			return err
		}

		if ifaceConfig.Bond {
			addBond(ifaceConfig, pciAddressList)
		} else if len(pciAddressList) == 1 {
			portName := "port-" + ifaceConfig.Bridge + "-1"
			addPort(ifaceConfig.Bridge, portName, pciAddressList[0])
		} else {
			err = fmt.Errorf("Multiple PCI address provided without bond: %s", pciAddressList)
			glog.Errorf("PrepareOvSBridgeConfig: %v", err)
			return err
		}
	}
	return nil
}

func GetPciAddressList(nicSelector ovsdpdkv1.NicSelector) ([]string, error) {
	pciAddressList := nicSelector.Devices
	if len(pciAddressList) == 0 {
		for _, ifName := range nicSelector.IfNames {
			name, err := getInterfacePciAddress(ifName)
			if err != nil {
				glog.Errorf("GetPciAddressList: failed: %v", err)
				return nil, err
			}
			pciAddressList = append(pciAddressList, name)
		}
	}
	return pciAddressList, nil
}

func bindDriver(driver string, pciAddressList []string) error {
	if driver == "" {
		glog.Infof("bindDriver: No drivers specified to apply bind")
		return nil
	}
	for _, pciAddress := range pciAddressList {
		err := BindDriver(pciAddress, driver)
		if err != nil {
			glog.Errorf("bindDriver: Failed to bind driver for device %s: %v", pciAddress, err)
			// TODO: (skramaja) Reverting of other devices?
			return err
		}
	}
	return nil
}

func getInterfacePciAddress(ifaceName string) (string, error) {
	devPath := path.Join(SYS_CLASS_NET, ifaceName, "device")
	_, err := os.Lstat(devPath)
	if err != nil {
		glog.Errorf("getInterfacePciAddress: Failed: %v", err)
		return "", err
	}

	devLink, err := os.Readlink(devPath)
	if err != nil {
		glog.Errorf("getInterfacePciAddress: Failed: %v", err)
		return "", err
	}

	_, f := path.Split(devLink)
	return f, nil
}

func addBridge(bridgeName string) error {
	return Run("ovs-vsctl", "--may-exist", "add-br", bridgeName, "--", "set", "bridge", bridgeName, "datapath_type=netdev")
}

func addBond(ifaceConfig ovsdpdkv1.InterfaceConfig, pciAddressList []string) error {
	err := fmt.Errorf("DPDK bond is implemented yet")
	glog.Errorf("addBond: %v", err)
	return err
}

func addPort(bridgeName, portName, pciAddress string) error {
	return Run("ovs-vsctl", "--may-exist", "add-port", bridgeName, portName, "--",
		"set", "Interface", portName, "type=dpdk", "options:dpdk-devargs="+pciAddress)
}
