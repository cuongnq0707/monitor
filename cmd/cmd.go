package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"monitor/network"
	"monitor/process"
	"monitor/send"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/spf13/cobra"
)

type Cmd struct {
	root      bool
	process   bool
	network   bool
	send_file bool
}

func NewCmd() *Cmd {
	return &Cmd{
		root:      true,
		process:   true,
		network:   true,
		send_file: true}
}
func (cmd *Cmd) Run() {
	rootCmd := &cobra.Command{
		Use:   "sysmon",
		Short: "System Monitor CLI",
		Long:  "A command-line tool to monitor system resources",
		Run:   cmd.rootCmdFunc,
	}
	if cmd.network {
		networkCmd := &cobra.Command{
			Use:   "network",
			Short: "network per processes",
			Long:  "A command-line tool to monitor network",
			Run:   cmd.networkCmd,
		}
		rootCmd.AddCommand(networkCmd)

	}
	if cmd.process {
		fmt.Println("process")
		processCmd := &cobra.Command{
			Use:   "processes",
			Short: "running processes, and threads",
			Long:  "A command-line tool to monitor running processes, and threads",
			Run:   cmd.processCmd,
		}
		rootCmd.AddCommand(processCmd)
	}
	if cmd.send_file {
		sftpCmd := &cobra.Command{
			Use:   "sftp",
			Short: "send file to sftp",
			Long:  "A command-line tool to send files to sftp",
			Run:   cmd.sendCmd,
		}
		rootCmd.AddCommand(sftpCmd)

	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func (_cmd *Cmd) processCmd(cmd *cobra.Command, args []string) {
	processMonitor := process.NewProcessMonitor()
	processMonitor.Start()
	processMonitor.Wait()
}
func (_cmd *Cmd) networkCmd(cmd *cobra.Command, args []string) {
	networkMonitor := network.NewNetworkMonitor()
	networkMonitor.Start()
	networkMonitor.Wait()
}
func (_cmd *Cmd) sendCmd(cmd *cobra.Command, args []string) {
	sftp_send := send.NewSftp("localhost", 22, "admin", "admin")
	sftp_send.Start()
}

func (_cmd *Cmd) rootCmdFunc(cmd *cobra.Command, args []string) {
	for {
		// Get CPU usage
		cpuPercent, err := cpu.Percent(time.Second, false)
		if err != nil {
			log.Fatal(err)
		}

		// Get memory usage
		memInfo, err := mem.VirtualMemory()
		if err != nil {
			log.Fatal(err)
		}

		// Get disk usage
		diskUsage, err := disk.Usage("/")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("CPU: %.2f%%, Memory: %.2f%%, Disk: %.2f%% \n", cpuPercent[0], memInfo.UsedPercent, diskUsage.UsedPercent)

	}
}
