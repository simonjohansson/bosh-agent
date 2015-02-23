package disk_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	. "github.com/cloudfoundry/bosh-agent/platform/disk"
	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
)

const devSdaSfdiskEmptyDump = `# partition table of /dev/sda
unit: sectors

/dev/sda1 : start=        0, size=    0, Id= 0
/dev/sda2 : start=        0, size=    0, Id= 0
/dev/sda3 : start=        0, size=    0, Id= 0
/dev/sda4 : start=        0, size=    0, Id= 0
`

const devSdaSfdiskNotableDumpStderr = `
sfdisk: ERROR: sector 0 does not have an msdos signature
 /dev/sda: unrecognized partition table type
No partitions found`

const devSdaSfdiskDump = `# partition table of /dev/sda
unit: sectors

/dev/sda1 : start=        1, size= xxxx, Id=82
/dev/sda2 : start=     xxxx, size= xxxx, Id=83
/dev/sda3 : start=     xxxx, size= xxxx, Id=83
/dev/sda4 : start=        0, size=    0, Id= 0
`

const devSdaSfdiskDumpOnePartition = `# partition table of /dev/sda
unit: sectors

/dev/sda1 : start=        1, size= xxxx, Id=83
/dev/sda2 : start=     xxxx, size= xxxx, Id=83
/dev/sda3 : start=        0, size=    0, Id= 0
/dev/sda4 : start=        0, size=    0, Id= 0
`

var _ = Describe("sfdiskPartitioner", func() {
	var (
		runner      *fakesys.FakeCmdRunner
		partitioner Partitioner
	)

	BeforeEach(func() {
		runner = fakesys.NewFakeCmdRunner()
		logger := boshlog.NewLogger(boshlog.LevelNone)

		partitioner = NewSfdiskPartitioner(logger, runner)
	})

	It("sfdisk partition", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskEmptyDump})

		partitions := []Partition{
			{Type: PartitionTypeSwap, SizeInBytes: 512 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 512 * 1024 * 1024},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(1).To(Equal(len(runner.RunCommandsWithInput)))
		Expect(runner.RunCommandsWithInput[0]).To(Equal([]string{"64,1048576,S\n1048704,2097152,L\n3145920,,L\n", "sfdisk", "-uS", "/dev/sda"}))
	})

	It("sfdisk partition with no partition table", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stderr: devSdaSfdiskNotableDumpStderr})

		partitions := []Partition{
			{Type: PartitionTypeSwap, SizeInBytes: 512 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 512 * 1024 * 1024},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(1).To(Equal(len(runner.RunCommandsWithInput)))
		Expect(runner.RunCommandsWithInput[0]).To(Equal([]string{"64,1048576,S\n1048704,2097152,L\n3145920,,L\n", "sfdisk", "-uS", "/dev/sda"}))
	})

	It("sfdisk get device size in mb", func() {
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 40000*1024)})

		size, err := partitioner.GetDeviceSizeInBytes("/dev/sda")
		Expect(err).ToNot(HaveOccurred())

		Expect(size).To(Equal(uint64(40000 * 1024 * 1024)))
	})

	It("sfdisk partition when partitions already match", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskDump})
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 2048*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda1", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 525*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda2", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1020*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda3", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 500*1024)})

		partitions := []Partition{
			{Type: PartitionTypeSwap, SizeInBytes: 512 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux, SizeInBytes: 512 * 1024 * 1024},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(len(runner.RunCommandsWithInput)).To(Equal(0))
	})

	It("sfdisk partition with last partition not matching size", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskDumpOnePartition})
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 2048*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda1", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1024*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda2", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 512*1024)})

		partitions := []Partition{
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(len(runner.RunCommandsWithInput)).To(Equal(1))
		Expect(runner.RunCommandsWithInput[0]).To(Equal([]string{"64,2097152,L\n2097280,,L\n", "sfdisk", "-uS", "/dev/sda"}))
	})

	It("sfdisk partition with last partition filling disk", func() {
		runner.AddCmdResult("sfdisk -d /dev/sda", fakesys.FakeCmdResult{Stdout: devSdaSfdiskDumpOnePartition})
		runner.AddCmdResult("sfdisk -s /dev/sda", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 2048*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda1", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1024*1024)})
		runner.AddCmdResult("sfdisk -s /dev/sda2", fakesys.FakeCmdResult{Stdout: fmt.Sprintf("%d\n", 1024*1024)})

		partitions := []Partition{
			{Type: PartitionTypeLinux, SizeInBytes: 1024 * 1024 * 1024},
			{Type: PartitionTypeLinux},
		}

		partitioner.Partition("/dev/sda", partitions)

		Expect(0).To(Equal(len(runner.RunCommandsWithInput)))
	})
})
