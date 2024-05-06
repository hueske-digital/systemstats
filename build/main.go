package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"sync"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type SystemInfo struct {
	Hostname  string  `json:"hostname"`
	RAMUsage  float64 `json:"ramUsage"`
	SwapUsage float64 `json:"swapUsage"`
	DiskUsage float64 `json:"diskUsage"`
}

func getSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}
	var wg sync.WaitGroup
	wg.Add(2)

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Error getting hostname: %v", err)
		hostname = "unknown"
	}
	info.Hostname = hostname

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
		info.RAMUsage = math.Round(virtualMem.UsedPercent)
		if swapMem.Total > 0 {
			info.SwapUsage = math.Round(float64(swapMem.Used) / float64(swapMem.Total) * 100)
		}
	}()

	go func() {
		defer wg.Done()
		diskStat, err := disk.Usage("/")
		if err != nil {
			log.Printf("Error getting disk: %v", err)
			return
		}
		info.DiskUsage = math.Round(diskStat.UsedPercent)
	}()

	wg.Wait()
	return info, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	info, err := getSystemInfo()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Printf("Error while encoding: %v", err)
		http.Error(w, "Error ", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
