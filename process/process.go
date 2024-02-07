package process

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/process"
)

// ProcessInfo stores information about a running process
type ProcessInfo struct {
	Name       string
	PID        int32
	CPUPercent float64
	MemPercent float32
	Threads    int32
	Networks   []NetworkInfo
}

type NetworkInfo struct {
	LocalIPAddress  string
	LocalPort       uint32
	RemoteIPAddress string
	RemotePort      uint32
	Status          string
}

type Option int

const (
	Exit Option = iota
	SortByCPU
	SortByMemory
	SortByName
	FindProcess
)

type ProcessMonitor struct {
	processInfoMap sync.Map
	stopSort       bool
	stopFilter     bool
	sortMutex      sync.Mutex
	filterMutex    sync.Mutex
	stopWG         sync.WaitGroup
	exitChan       chan struct{}
	exitProcess    chan struct{}
	stopProcess    bool
}

func NewProcessMonitor() *ProcessMonitor {
	return &ProcessMonitor{
		processInfoMap: sync.Map{},
		stopSort:       false,
		stopFilter:     false,
		sortMutex:      sync.Mutex{},
		filterMutex:    sync.Mutex{},
		exitChan:       make(chan struct{}),
		exitProcess:    make(chan struct{}),
		stopProcess:    false,
	}
}

func (pm *ProcessMonitor) Wait() {
	<-pm.exitChan
}
func (pm *ProcessMonitor) Start() {
	go pm.getUserInput()
	go pm.updateProcessInfo()
}

func (pm *ProcessMonitor) Stop() {

	close(pm.exitChan)
	close(pm.exitProcess)
}

func (pm *ProcessMonitor) getUserInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Enter option to sort. 0: exit, 1: CPU usage, 2: Memory usage, 3: Name, 4: Find process => ")
		if !scanner.Scan() {
			fmt.Println("Error reading input:", scanner.Err())
			continue
		}
		optionStr := scanner.Text()
		if len(optionStr) == 0 {
			continue
		}
		userSelected, err := strconv.Atoi(strings.TrimSpace(optionStr))
		if err != nil || userSelected < 0 || userSelected > int(FindProcess) {
			fmt.Println("Invalid input. Please enter a valid option.")
			continue
		}
		if pm.stopProcess == true {
			pm.exitProcess <- struct{}{}
		}
		if Option(userSelected) == Exit {
			pm.Stop()
			break
		} else if Option(userSelected) == FindProcess {
			fmt.Println("")
			fmt.Print("Enter process name to filter => ")
			if !scanner.Scan() {
				fmt.Println("Error reading input:", scanner.Err())
				continue
			}
			processName := scanner.Text()
			processName = strings.TrimSpace(processName)
			go pm.filterProcesses(processName)
		} else {
			go pm.sortProcesses(Option(userSelected))
		}
		pm.stopProcess = true
	}
}

func (pm *ProcessMonitor) sortProcesses(option Option) {
	for {
		select {
		case <-pm.exitProcess:
			fmt.Println("Exited sortProcesses")
			return
		default:
			processes := pm.getProcessInfoMap()

			sortedProcesses := make([]ProcessInfo, 0, len(processes))
			for _, pInfo := range processes {
				sortedProcesses = append(sortedProcesses, pInfo)
			}

			switch option {
			case SortByCPU:
				// Sort processes by CPU usage
				sort.Slice(sortedProcesses, func(i, j int) bool {
					return sortedProcesses[i].CPUPercent > sortedProcesses[j].CPUPercent
				})
			case SortByMemory:
				// Sort processes by memory usage
				sort.Slice(sortedProcesses, func(i, j int) bool {
					return sortedProcesses[i].MemPercent > sortedProcesses[j].MemPercent
				})
			case SortByName:
				// Sort processes by name
				sort.Slice(sortedProcesses, func(i, j int) bool {
					return sortedProcesses[i].Name < sortedProcesses[j].Name
				})
			}

			pm.printProcessInfo(sortedProcesses)
			time.Sleep(time.Second * 7)
		}
	}
}

func (pm *ProcessMonitor) filterProcesses(processName string) {
	for {
		select {
		case <-pm.exitProcess:
			fmt.Println("Exited filterProcesses")
			return
		default:
			processes := pm.getProcessInfoMap()
			filteredProcesses := make([]ProcessInfo, 0)
			for _, pInfo := range processes {
				if strings.Contains(pInfo.Name, processName) {
					filteredProcesses = append(filteredProcesses, pInfo)
				}
			}

			pm.printProcessInfo(filteredProcesses)
			time.Sleep(time.Second * 7)
		}
	}
}

func (pm *ProcessMonitor) updateProcessInfo() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-pm.exitChan:
			fmt.Println("Exited updateProcessInfo")
			return
		case <-ticker.C:

			processes, err := process.Processes()
			if err != nil {
				log.Println("Error retrieving running processes:", err)
				continue
			}

			pm.processInfoMap = sync.Map{}
			for _, p := range processes {
				name, _ := p.Name()
				pid := p.Pid
				cpuPercent, _ := p.CPUPercent()
				memPercent, _ := p.MemoryPercent()
				threads, _ := p.NumThreads()

				connections, _ := p.Connections()
				networks := make([]NetworkInfo, 0)
				for _, conn := range connections {
					networks = append(networks, NetworkInfo{
						LocalIPAddress:  conn.Laddr.IP,
						LocalPort:       conn.Laddr.Port,
						RemoteIPAddress: conn.Raddr.IP,
						RemotePort:      conn.Raddr.Port,
						Status:          conn.Status,
					})
				}

				processInfo := ProcessInfo{
					Name:       name,
					PID:        pid,
					CPUPercent: cpuPercent,
					MemPercent: memPercent,
					Threads:    threads,
					Networks:   networks,
				}
				pm.processInfoMap.Store(pid, processInfo)
			}
		}
	}
}

func (pm *ProcessMonitor) getProcessInfoMap() map[int32]ProcessInfo {
	processes := make(map[int32]ProcessInfo)

	pm.processInfoMap.Range(func(key, value interface{}) bool {
		pid := key.(int32)
		pInfo := value.(ProcessInfo)
		processes[pid] = pInfo
		return true
	})

	return processes
}

func (pm *ProcessMonitor) printProcessInfo(processes []ProcessInfo) {
	fmt.Println("-----------------------------------------------------------")
	fmt.Println("Name\t\tPID\tCPU%\tMemory%\tThreads\tLocal Address\tRemote Address")
	fmt.Println("-----------------------------------------------------------")
	for _, pInfo := range processes {
		fmt.Printf("%s\t%d\t%.2f\t%.2f\t%d\t", pInfo.Name, pInfo.PID, pInfo.CPUPercent, pInfo.MemPercent, pInfo.Threads)
		if len(pInfo.Networks) > 0 {
			for i, net := range pInfo.Networks {
				fmt.Printf("%s: %d\t %s: %d", net.LocalIPAddress, net.LocalPort, net.RemoteIPAddress, net.RemotePort)
				if i != len(pInfo.Networks)-1 {
					fmt.Print(", ")
				}
			}
		} else {
			fmt.Print("N/A")
		}
		fmt.Println()
	}
	fmt.Println("-----------------------------------------------------------")
}
