package prepare

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/golang/glog"
	iowrap "github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	ovsdpdkv1 "github.com/krsacme/ovsdpdk-network-operator/pkg/apis/ovsdpdknetwork/v1"
)

var (
	FS     iowrap.Fs
	FSUtil *iowrap.Afero
)

func init() {
	FS = iowrap.NewOsFs()
	FSUtil = &iowrap.Afero{Fs: FS}
}

func PrepareOvsDpdkConfig(kc *kubernetes.Clientset) error {
	cmName, found := os.LookupEnv("OVSDPDK_PREAPE_CONFIG_MAP")
	if !found {
		glog.Errorf("Failed to look up env OVSDPDK_PREAPE_CONFIG_MAP")
		return fmt.Errorf("Failed to lookup env OVSDPDK_PREAPE_CONFIG_MAP")
	}

	namespace, found := os.LookupEnv("NAMESPACE")
	if !found {
		glog.Errorf("Failed to look up env NAMESPACE")
		return fmt.Errorf("Failed to lookup env NAMESPACE")
	}

	glog.Infof("Preparing the node for OvsDPDK configuration: Namespace(%s) ConfigMap(%s)", namespace, cmName)

	options := metav1.GetOptions{}
	cm, err := kc.CoreV1().ConfigMaps(namespace).Get(context.TODO(), cmName, options)
	if err != nil {
		glog.Errorf("Failed to get the ConfigMap resource: %v", err)
		return err
	}

	ifaceJson := cm.Data["interface"]
	glog.Infof("Interface JSON: %s", ifaceJson)
	ifaceConfig := &[]ovsdpdkv1.InterfaceConfig{}
	err = json.Unmarshal([]byte(ifaceJson), ifaceConfig)
	if err != nil {
		glog.Errorf("Failed to unmarshal interface config: %v", err)
		return err
	}

	nodeJson := cm.Data["node"]
	glog.Infof("Node Json: %s", nodeJson)
	nodeConfig := &ovsdpdkv1.NodeConfig{}
	err = json.Unmarshal([]byte(nodeJson), nodeConfig)
	if err != nil {
		glog.Errorf("Failed to unmarshal node config: %v", err)
		return err
	}

	// Kernel Ars - intel_iommu=on iommu=pt
	// Ensure Hugepages is available
	// TODO: (skramaja) Hugpage and reboot

	// Step 2 - Configure OvS-DPDK Config and Enable DPDK
	err = PrepareOvSDPDKConfig(nodeConfig, ifaceConfig)
	if err != nil {
		glog.Errorf("Failed to configure DPDK, exiting..")
		return err
	}

	// Step 3 - Create OvS Bridges and DPDK ports
	err = PrepareOvsBridgeConfig(ifaceConfig)
	if err != nil {
		glog.Errorf("Failed to configure Bridge, exiting..")
		return err
	}

	return nil
}
