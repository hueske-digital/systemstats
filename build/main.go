package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"sync"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type SystemInfo struct {
	RAMUsagePercent  float64 `json:"ramUsagePercent"`
	SwapUsagePercent float64 `json:"swapUsagePercent"`
	DiskUsagePercent float64 `json:"diskUsagePercent"`
	Load1            float64 `json:"load1"`
	Load5            float64 `json:"load5"`
	Load15           float64 `json:"load15"`
	NetworkIn        float64 `json:"networkIn"`
	NetworkOut       float64 `json:"networkOut"`
}

func getSystemInfo(netInterface string) (*SystemInfo, error) {
	info := &SystemInfo{}
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		virtualMem, err := mem.VirtualMemory()
		if err != nil {
			log.Printf("Error getting memory: %v", err)
			return
		}
		swapMem, err := mem.SwapMemory()
		if err != nil {
			log.Printf("Error getting swap: %v", err)
			return
		}
		info.RAMUsagePercent = math.Round(virtualMem.UsedPercent)
		if swapMem.Total > 0 {
			info.SwapUsagePercent = math.Round(float64(swapMem.Used) / float64(swapMem.Total) * 100)
		}
	}()

	go func() {
		defer wg.Done()
		diskStat, err := disk.Usage("/")
		if err != nil {
			log.Printf("Error getting disk: %v", err)
			return
		}
		info.DiskUsagePercent = math.Round(diskStat.UsedPercent)
	}()

	go func() {
		defer wg.Done()
		loadStat, err := load.Avg()
		if err != nil {
			log.Printf("Error getting load averages: %v", err)
			return
		}
		info.Load1 = math.Round(loadStat.Load1)
		info.Load5 = math.Round(loadStat.Load5)
		info.Load15 = math.Round(loadStat.Load15)
	}()

	go func() {
		defer wg.Done()
		counters, err := net.IOCounters(true)
		if err != nil {
			log.Printf("Error getting network counters: %v", err)
			return
		}
		for _, counter := range counters {
			if counter.Name == netInterface {
				info.NetworkIn = math.Round(float64(counter.BytesRecv) / (1024 * 1024))
				info.NetworkOut = math.Round(float64(counter.BytesSent) / (1024 * 1024))
				break
			}
		}
	}()

	wg.Wait()
	return info, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	netInterface := os.Getenv("NET_INTERFACE")
	if netInterface == "" {
		netInterface = "eth0"
	}

	info, err := getSystemInfo(netInterface)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Printf("Error while encoding: %v", err)
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
