package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aeden/traceroute"
	"github.com/go-ping/ping"
)

type Option int

const (
	Exit Option = iota
	ScanPort
	PingHost
	TraceRoute
)

type NetworkMonitor struct {
	openPorts   []int32
	exitChan    chan struct{}
	exitProcess chan struct{}
}

func NewNetworkMonitor() *NetworkMonitor {
	return &NetworkMonitor{exitChan: make(chan struct{}), exitProcess: make(chan struct{})}
}

func (nm *NetworkMonitor) scanPort(host string, port int, wg *sync.WaitGroup) {
	defer wg.Done()

	address := host + ":" + strconv.Itoa(port)
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err == nil {
		fmt.Printf("Port %d is open\n", port)
		conn.Close()
	}
}
func (nm *NetworkMonitor) Wait() {
	<-nm.exitChan
}
func (nm *NetworkMonitor) Stop() {
	close(nm.exitChan)
	close(nm.exitProcess)
}
func (nm *NetworkMonitor) Start() {
	fmt.Println("NetworkMonitor Start")
	go nm.getUserInput()
}
func (nm *NetworkMonitor) getUserInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Enter option to sort. 0: exit, 1: scan port , 2: ping host , 3: traceroute => ")
		if !scanner.Scan() {
			fmt.Println("Error reading input:", scanner.Err())
			continue
		}
		optionStr := scanner.Text()
		if len(optionStr) == 0 {
			continue
		}
		userSelected, err := strconv.Atoi(strings.TrimSpace(optionStr))
		if err != nil || userSelected < 0 || userSelected > int(TraceRoute) {
			fmt.Println("Invalid input. Please enter a valid option.")
			continue
		}
		switch Option(userSelected) {
		case Exit:
			nm.Stop()
			break
		case PingHost:
		case TraceRoute:
			fmt.Println("")
			fmt.Print("Enter host => ")
			if !scanner.Scan() {
				fmt.Println("Error reading input:", scanner.Err())
				continue
			}
			hostName := scanner.Text()
			hostName = strings.TrimSpace(hostName)
			if Option(userSelected) == PingHost {
				go nm.pingHost(hostName)
			} else if Option(userSelected) == TraceRoute {
				go nm.traceRoute(hostName)
			}
		}
	}
}
func (nm *NetworkMonitor) scan(host string) {
	startPort := 1
	endPort := 65535

	var wg sync.WaitGroup

	// Scan for open ports
	for port := startPort; port <= endPort; port++ {
		wg.Add(1)
		go nm.scanPort(host, port, &wg)
	}

	wg.Wait()
}
func (nm *NetworkMonitor) pingHost(host string) {
	for {
		select {
		case <-nm.exitProcess:
			fmt.Println("Exit pinghost")
			return
		default:
			pinger, err := ping.NewPinger(host)
			if err != nil {
				panic(err)
			}

			// Set the number of packets to send
			pinger.Count = 3

			// Start the pinger
			err = pinger.Run()
			if err != nil {
				panic(err)
			}

			// Print the statistics after pinging is finished
			stats := pinger.Statistics()
			fmt.Printf("Packets transmitted: %d, packets received: %d, packet loss: %f%%\n",
				stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)

			// Listen for interrupt signal to stop the pinger
			pinger.Stop()
		}
		time.Sleep(time.Second * 4)
	}
}

func (nm *NetworkMonitor) printHop(hop traceroute.TracerouteHop) {
	addr := fmt.Sprintf("%v.%v.%v.%v", hop.Address[0], hop.Address[1], hop.Address[2], hop.Address[3])
	hostOrAddr := addr
	if hop.Host != "" {
		hostOrAddr = hop.Host
	}
	if hop.Success {
		fmt.Printf("%-3d %v (%v)  %v\n", hop.TTL, hostOrAddr, addr, hop.ElapsedTime)
	} else {
		fmt.Printf("%-3d *\n", hop.TTL)
	}
}
func (nm *NetworkMonitor) traceRoute(destination string) {

	options := traceroute.TracerouteOptions{}
	// Resolve the destination IP address
	_, err := net.ResolveIPAddr("ip", destination)
	if err != nil {
		log.Fatal("Failed to resolve IP address:", err)
	}

	c := make(chan traceroute.TracerouteHop, 0)
	go func() {
		for {
			hop, ok := <-c
			if !ok {
				fmt.Println()
				return
			}
			nm.printHop(hop)
		}
	}()

	_, err = traceroute.Traceroute(destination, &options, c)
	if err != nil {
		fmt.Printf("Error: ", err)
	}
}
