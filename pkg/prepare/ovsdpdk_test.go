package prepare_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	iowrap "github.com/spf13/afero"

	ovsdpdkv1 "github.com/krsacme/ovsdpdk-network-operator/pkg/apis/ovsdpdk/v1"
	. "github.com/krsacme/ovsdpdk-network-operator/pkg/prepare"
)

const (
	NODE_FILE = "/sys/devices/system/node/online"

	NODE0 = "/sys/devices/system/node/node0/cpulist"
	NODE1 = "/sys/devices/system/node/node1/cpulist"
	NODE2 = "/sys/devices/system/node/node2/cpulist"
	NODE3 = "/sys/devices/system/node/node3/cpulist"

	PCI0 = SYS_BUS_PCI_DEVICES + "0000:01:00.0/numa_node"
	PCI1 = SYS_BUS_PCI_DEVICES + "0000:02:00.0/numa_node"
	PCI2 = SYS_BUS_PCI_DEVICES + "0000:03:00.0/numa_node"
	PCI3 = SYS_BUS_PCI_DEVICES + "0000:04:00.0/numa_node"

	CPU0  = "/sys/devices/system/cpu/cpu0/topology/thread_siblings"
	CPU1  = "/sys/devices/system/cpu/cpu1/topology/thread_siblings"
	CPU2  = "/sys/devices/system/cpu/cpu2/topology/thread_siblings"
	CPU3  = "/sys/devices/system/cpu/cpu3/topology/thread_siblings"
	CPU4  = "/sys/devices/system/cpu/cpu4/topology/thread_siblings"
	CPU5  = "/sys/devices/system/cpu/cpu5/topology/thread_siblings"
	CPU6  = "/sys/devices/system/cpu/cpu6/topology/thread_siblings"
	CPU7  = "/sys/devices/system/cpu/cpu7/topology/thread_siblings"
	CPU8  = "/sys/devices/system/cpu/cpu8/topology/thread_siblings"
	CPU9  = "/sys/devices/system/cpu/cpu9/topology/thread_siblings"
	CPU10 = "/sys/devices/system/cpu/cpu10/topology/thread_siblings"
	CPU11 = "/sys/devices/system/cpu/cpu11/topology/thread_siblings"
)

func init() {
	FS = iowrap.NewMemMapFs()
	FSUtil = &iowrap.Afero{Fs: FS}

	iowrap.WriteFile(FS, PCI0, []byte("0\n"), 0644)
	iowrap.WriteFile(FS, PCI1, []byte("1\n"), 0644)
	iowrap.WriteFile(FS, PCI2, []byte("2\n"), 0644)
	iowrap.WriteFile(FS, PCI3, []byte("3\n"), 0644)
}

var _ = Describe("Pkg/Prepare/Ovsdpdk", func() {
	Describe("Get CPU Mask", func() {
		Context("Mask", func() {
			It("should get mask for cpu id 0", func() {
				Expect(GetCpuMask([]int{0})).To(Equal("1"))
			})
			It("should get mask for cpu id 1", func() {
				Expect(GetCpuMask([]int{1})).To(Equal("2"))
			})
			It("should get mask for cpu id 0,1", func() {
				Expect(GetCpuMask([]int{0, 1})).To(Equal("3"))
			})
			It("should get mask for cpu id 0,1,2,3", func() {
				Expect(GetCpuMask([]int{0, 1, 2, 3})).To(Equal("f"))
			})
			It("should get mask for cpu id 4", func() {
				Expect(GetCpuMask([]int{4})).To(Equal("10"))
			})
			It("should get mask for cpu id 0,1,22,23", func() {
				Expect(GetCpuMask([]int{0, 22, 1, 23})).To(Equal("c00003"))
			})
			It("should get mask for cpu id 0,1,64,65", func() {
				Expect(GetCpuMask([]int{0, 64, 65, 1})).To(Equal("30000000000000003"))
			})
			It("should get mask for cpu id 64", func() {
				Expect(GetCpuMask([]int{64})).To(Equal("10000000000000000"))
			})
			It("should get mask for cpu id 65", func() {
				Expect(GetCpuMask([]int{65})).To(Equal("20000000000000000"))
			})
		})
	})

	Describe("Get cores", func() {
		Context("Single NUMA", func() {
			BeforeEach(func() {
				iowrap.WriteFile(FS, NODE_FILE, []byte("0\n"), 0644)
				iowrap.WriteFile(FS, NODE0, []byte("0-7\n"), 0644)
				iowrap.WriteFile(FS, CPU0, []byte("0,4\n"), 0644)
				iowrap.WriteFile(FS, CPU1, []byte("1,5\n"), 0644)
				iowrap.WriteFile(FS, CPU2, []byte("2,6\n"), 0644)
				iowrap.WriteFile(FS, CPU3, []byte("3,7\n"), 0644)
			})
			It("Should get Lcore cpus", func() {
				pmdCpus, err := GetLcores()
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{0, 4}))
			})
			It("Should get 1 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 1,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5}))
			})
			It("Should get 2 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 2,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5}))
			})
			It("Should get 4 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 4,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5, 2, 6}))
			})
			It("Should get 4 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 6,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5, 2, 6, 3, 7}))
			})
		})
		Context("Dual NUMA with DPDK interface in NUMA0", func() {
			BeforeEach(func() {
				iowrap.WriteFile(FS, NODE_FILE, []byte("0,1\n"), 0644)
				iowrap.WriteFile(FS, NODE0, []byte("0-7\n"), 0644)
				iowrap.WriteFile(FS, CPU0, []byte("0,4\n"), 0644)
				iowrap.WriteFile(FS, CPU1, []byte("1,5\n"), 0644)
				iowrap.WriteFile(FS, CPU2, []byte("2,6\n"), 0644)
				iowrap.WriteFile(FS, CPU3, []byte("3,7\n"), 0644)
				iowrap.WriteFile(FS, NODE1, []byte("8-15\n"), 0644)
				iowrap.WriteFile(FS, CPU8, []byte("8,12\n"), 0644)
				iowrap.WriteFile(FS, CPU9, []byte("9,13\n"), 0644)
				iowrap.WriteFile(FS, CPU10, []byte("10,14\n"), 0644)
				iowrap.WriteFile(FS, CPU11, []byte("11,15\n"), 0644)
			})
			It("Should get Lcore cpus", func() {
				pmdCpus, err := GetLcores()
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{0, 4, 8, 12}))
			})
			It("Should get 1 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 1,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5, 9, 13}))
			})
			It("Should get 2 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 2,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5, 9, 13}))
			})
			It("Should get 4 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 4,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5, 2, 6, 9, 13}))
			})
		})
		Context("Dual NUMA with DPDK interface in NUMA1", func() {
			BeforeEach(func() {
				iowrap.WriteFile(FS, NODE_FILE, []byte("0,1\n"), 0644)
				iowrap.WriteFile(FS, NODE0, []byte("0-7\n"), 0644)
				iowrap.WriteFile(FS, CPU0, []byte("0,4\n"), 0644)
				iowrap.WriteFile(FS, CPU1, []byte("1,5\n"), 0644)
				iowrap.WriteFile(FS, CPU2, []byte("2,6\n"), 0644)
				iowrap.WriteFile(FS, CPU3, []byte("3,7\n"), 0644)
				iowrap.WriteFile(FS, NODE1, []byte("8-15\n"), 0644)
				iowrap.WriteFile(FS, CPU8, []byte("8,12\n"), 0644)
				iowrap.WriteFile(FS, CPU9, []byte("9,13\n"), 0644)
				iowrap.WriteFile(FS, CPU10, []byte("10,14\n"), 0644)
				iowrap.WriteFile(FS, CPU11, []byte("11,15\n"), 0644)
			})
			It("Should get Lcore cpus", func() {
				pmdCpus, err := GetLcores()
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{0, 4, 8, 12}))
			})
			It("Should get 1 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 1,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:02:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{9, 13, 1, 5}))
			})
			It("Should get 2 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 2,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:02:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{9, 13, 1, 5}))
			})
			It("Should get 4 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 4,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:02:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{9, 13, 10, 14, 1, 5}))
			})
		})
		Context("Dual NUMA with DPDK interface both NUMA", func() {
			BeforeEach(func() {
				iowrap.WriteFile(FS, NODE_FILE, []byte("0,1\n"), 0644)
				iowrap.WriteFile(FS, NODE0, []byte("0-7\n"), 0644)
				iowrap.WriteFile(FS, CPU0, []byte("0,4\n"), 0644)
				iowrap.WriteFile(FS, CPU1, []byte("1,5\n"), 0644)
				iowrap.WriteFile(FS, CPU2, []byte("2,6\n"), 0644)
				iowrap.WriteFile(FS, CPU3, []byte("3,7\n"), 0644)
				iowrap.WriteFile(FS, NODE1, []byte("8-15\n"), 0644)
				iowrap.WriteFile(FS, CPU8, []byte("8,12\n"), 0644)
				iowrap.WriteFile(FS, CPU9, []byte("9,13\n"), 0644)
				iowrap.WriteFile(FS, CPU10, []byte("10,14\n"), 0644)
				iowrap.WriteFile(FS, CPU11, []byte("11,15\n"), 0644)
			})
			It("Should get 1 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 1,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0", "0000:02:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5, 9, 13}))
			})
			It("Should get 4 PMD cpu", func() {
				nodeConfig := ovsdpdkv1.NodeConfig{
					PMDCount: 4,
				}
				pmdCpus, err := GetPmdCpus(nodeConfig, []string{"0000:01:00.0", "0000:02:00.0"})
				Expect(err).NotTo(HaveOccurred())
				Expect(pmdCpus).To(Equal([]int{1, 5, 2, 6, 9, 13, 10, 14}))
			})
		})
	})
})
