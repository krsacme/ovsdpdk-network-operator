package prepare

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"

	ovsdpdkv1 "github.com/krsacme/ovsdpdk-network-operator/pkg/apis/ovsdpdknetwork/v1"
)

const (
	SYS_DEVICES_SYSTEM = SYS + "devices/system/"
)

func PrepareOvSDPDKConfig(nodeConfig *ovsdpdkv1.NodeConfig, ifaceConfig *[]ovsdpdkv1.InterfaceConfig) error {
	var pciAddressList []string
	for _, cfg := range *ifaceConfig {
		pci, err := GetPciAddressList(cfg.NicSelector)
		if err != nil {
			glog.Errorf("PrepareOvSDPDKConfig: Failed to get PCI address list: %v", err)
			return err
		}
		pciAddressList = append(pciAddressList, pci...)
	}

	// PMD CPUs
	pmd, err := GetPmdCpus(nodeConfig, pciAddressList)
	if err != nil {
		glog.Errorf("PrepareOvSDPDKConfig: failed to get PMD Cpus: %v", err)
		return err
	}
	pmdMask := GetCpuMask(pmd)
	glog.Infof("PrepareOvSDPDKConfig: PMD CPU list (%v) PMD CPU mask (%s)", pmd, pmdMask)
	err = Run("ovs-vsctl", "set", "Open_vSwitch", ".", "other_config:pmd-cpu-mask=0x"+pmdMask)
	if err != nil {
		return err
	}

	// Memory Channel
	// Setting default memory channel to 4, it depends on the system architecture
	// TODO: (skramaja) Is there a posiblity to create a map of all types of system to support?
	err = Run("ovs-vsctl", "set", "Open_vSwitch", ".", "other_config:dpdk-extra=\"-n 4 \"")
	if err != nil {
		return err
	}

	// Socket Memory
	// TODO: (skramaja) Fix MTU value from config
	sockMem, err := getSocketMemory(9000, pciAddressList)
	err = Run("ovs-vsctl", "set", "Open_vSwitch", ".", "other_config:dpdk-socket-mem="+sockMem)
	if err != nil {
		return err
	}

	// LCores (?) Is it required?
	lcores, err := GetLcores()
	lcoreMask := GetCpuMask(lcores)
	glog.Infof("PrepareOvSDPDKConfig: LCORE list (%v) LCORE mask (%s)", lcores, lcoreMask)
	err = Run("ovs-vsctl", "set", "Open_vSwitch", ".", "other_config:dpdk-lcore-mask=0x"+lcoreMask)
	if err != nil {
		return err
	}

	// Revalidator and Handler threads (?) Is it required?

	// Enable DPDK
	err = Run("ovs-vsctl", "set", "Open_vSwitch", ".", "other_config:dpdk-init=true")
	if err != nil {
		return err
	}
	return nil
}

func Run(name string, arg ...string) error {
	glog.Infof("OvS Command: %s %s", name, arg)
	cmd := exec.Command(name, arg...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())

	if err != nil {
		glog.Errorf("Run: failed: %v", err)
		glog.Errorf("Run: sterr: %s", errStr)
		return err
	} else if outStr != "" {
		glog.Infof("Run: stdout: %s", outStr)
	}
	return nil
}

func getSocketMemory(mtu int, pciAddressList []string) (string, error) {
	// TODO: (skramaja) Implement socket memory algorithm
	return "1024,1024", nil
}

func GetLcores() ([]int, error) {
	numaNodes, err := getAllNumaNodes()
	if err != nil {
		glog.Errorf("GetLcores: Failed to get all NUMA nodes: %v", err)
		return nil, err
	}

	var lcores []int
	for _, numa := range numaNodes {
		cpulist, err := getNumaCpus(numa)
		if err != nil {
			glog.Errorf("GetLcores: Failed to get NUMA nodes cpus: %v", err)
			return nil, err
		}

		siblingPath := fmt.Sprintf(SYS_DEVICES_SYSTEM+"cpu/cpu%d/topology/thread_siblings", cpulist[0])
		firstCoreCpus, err := getIntContent(siblingPath)
		if err != nil {
			glog.Errorf("GetLcores: Failed to get thread siblings: %v", err)
			return nil, err
		}
		lcores = append(lcores, firstCoreCpus...)
	}
	return lcores, nil
}

func GetCpuMask(cpulist []int) string {
	sort.Ints(cpulist)
	binStr := make([]byte, cpulist[len(cpulist)-1]+1)
	for i := 0; i < len(binStr); i++ {
		binStr[i] = '0'
	}
	for i := 0; i < len(cpulist); i++ {
		binStr[len(binStr)-cpulist[i]-1] = '1'
	}
	size := len(binStr)
	idx := size
	out := ""
	for idx >= 0 {
		start := idx - 64
		if idx < 64 {
			start = 0
		}
		ui, _ := strconv.ParseUint(string(binStr[start:idx]), 2, 64)
		idx -= 64
		if idx >= 0 {
			out = fmt.Sprintf("%016x", ui) + out
		} else {
			out = fmt.Sprintf("%x", ui) + out
		}
	}
	return out
}

func GetPmdCpus(nodeConfig *ovsdpdkv1.NodeConfig, pciAddressList []string) ([]int, error) {
	ifaceNumaNodes, err := getInterfaceNumaNodes(pciAddressList)
	if err != nil {
		glog.Errorf("getPmdCpus: Failed to get NUMA nodes: %v", err)
		return nil, err
	}

	if len(ifaceNumaNodes) == 0 {
		// List will be empty when there is only one numa node
		ifaceNumaNodes = append(ifaceNumaNodes, 0)
	}

	var pmd []int
	for _, numa := range ifaceNumaNodes {
		cpus, err := getNumaPmdCpus(numa, int(nodeConfig.PMDCount))
		if err != nil {
			glog.Errorf("getPmdCpus: Failed to pmd cpus for numa %d: %v", numa, err)
			return nil, err
		}
		pmd = append(pmd, cpus...)
	}

	numaNodes, err := getAllNumaNodes()
	if err != nil {
		glog.Errorf("getPmdCpus: Failed to get all NUMA nodes: %v", err)
		return nil, err
	}
	nonIfaceNumaNodes := difference(numaNodes, ifaceNumaNodes)
	for _, numa := range nonIfaceNumaNodes {
		cpus, err := getNumaPmdCpus(numa, 1)
		if err != nil {
			glog.Errorf("getPmdCpus: Failed to pmd cpus for numa %d: %v", numa, err)
			return nil, err
		}
		pmd = append(pmd, cpus...)
	}
	return pmd, nil
}

func getNumaCpus(numa int) ([]int, error) {
	cpulistPath := fmt.Sprintf(SYS_DEVICES_SYSTEM+"node/node%d/cpulist", numa)
	cpulist, err := getIntContent(cpulistPath)
	if err != nil {
		glog.Errorf("getNumaCpus: Failed to get NUMA node cpulist: %v", err)
		return nil, err
	}
	sort.Ints(cpulist)
	return cpulist, nil
}

func getNumaPmdCpus(numa, pmdCount int) ([]int, error) {
	numaCpulist, err := getNumaCpus(numa)
	if err != nil {
		return nil, err
	}

	siblingPath := fmt.Sprintf(SYS_DEVICES_SYSTEM+"cpu/cpu%d/topology/thread_siblings", numaCpulist[0])
	firstCoreCpus, err := getIntContent(siblingPath)
	if err != nil {
		glog.Errorf("getNumaPmdCpus: Failed to get thread siblings: %v", err)
		return nil, err
	}
	numaCpulist = difference(numaCpulist, firstCoreCpus)

	var pmdCpus []int
	var cpu int = 0
	for i := 0; i < pmdCount; {
		siblingPath := fmt.Sprintf(SYS_DEVICES_SYSTEM+"cpu/cpu%d/topology/thread_siblings", numaCpulist[cpu])
		cpu++
		cpuList, err := getIntContent(siblingPath)
		if err != nil {
			glog.Errorf("getNumaPmdCpus: Failed to get thread siblings: %v", err)
			return nil, err
		}
		i += len(cpuList)
		pmdCpus = append(pmdCpus, cpuList...)
	}
	return pmdCpus, nil
}

func getIntContent(filePath string) ([]int, error) {
	content, err := FSUtil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	cpus := getIntArrayFromRange(string(content))
	return cpus, nil
}

func getInterfaceNumaNodes(pciAddressList []string) ([]int, error) {
	var numaNodes []int
	for _, pciAddress := range pciAddressList {
		pciNumaPath := path.Join(SYS_BUS_PCI_DEVICES, pciAddress, "numa_node")
		content, err := FSUtil.ReadFile(pciNumaPath)
		if err != nil {
			glog.Errorf("getNumaNodes: Failed to read file %s: %v", pciNumaPath, err)
			return nil, err
		}
		contentStr := strings.TrimSuffix(string(content), "\n")
		numa, _ := strconv.Atoi(contentStr)
		if numa >= 0 {
			numaNodes = append(numaNodes, int(numa))
		}
	}
	return numaNodes, nil
}

func getAllNumaNodes() ([]int, error) {
	content, err := FSUtil.ReadFile(SYS_DEVICES_SYSTEM + "node/online")
	if err != nil {
		glog.Errorf("getAllNumaNodes: Failed to get numa nodes: %v", err)
		return nil, err
	}
	numaList := string(content)
	return getIntArrayFromRange(numaList), nil
}

func getIntArrayFromRange(content string) []int {
	var result []int
	content = strings.TrimSuffix(content, "\n")
	parts := strings.Split(content, ",")
	for _, item := range parts {
		if strings.Contains(item, "-") {
			itemParts := strings.Split(item, "-")
			start, _ := strconv.Atoi(itemParts[0])
			end, _ := strconv.Atoi(itemParts[1])
			for i := int(start); i <= int(end); i++ {
				result = append(result, i)
			}
		} else {
			i, _ := strconv.Atoi(item)
			result = append(result, int(i))
		}
	}
	return result
}

func difference(a, b []int) []int {
	target := map[int]bool{}
	for _, x := range b {
		target[x] = true
	}

	result := []int{}
	for _, x := range a {
		if _, ok := target[x]; !ok {
			result = append(result, x)
		}
	}

	return result
}
