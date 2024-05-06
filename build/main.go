package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

type SystemInfo struct {
	RAMUsagePercent    float64 `json:"ramUsagePercent"`
	SwapUsagePercent   float64 `json:"swapUsagePercent"`
	DiskUsagePercent   float64 `json:"diskUsagePercent"`
	CPUUsagePercent    float64 `json:"cpuUsagePercent"`
	TrafficUsedPercent float64 `json:"trafficUsedPercent"`
}

var (
	trafficCache      float64
	lastTrafficUpdate time.Time
	cacheLock         sync.Mutex
)

func getSystemInfo(client *hcloud.Client, serverID int) (*SystemInfo, error) {
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
		numCores, err := cpu.Counts(true)
		if err != nil {
			log.Printf("Error getting CPU cores: %v", err)
			return
		}
		info.CPUUsagePercent = math.Round(loadStat.Load1 / float64(numCores) * 100)
	}()

	go func() {
		defer wg.Done()
		cacheLock.Lock()
		defer cacheLock.Unlock()
		if time.Since(lastTrafficUpdate) < 60*time.Second {
			info.TrafficUsedPercent = trafficCache
			return
		}

		server, _, err := client.Server.GetByID(context.Background(), int64(serverID))
		if err != nil {
			log.Printf("Error getting server data from Hetzner: %v", err)
			return
		}
		if server != nil {
			includedTraffic := server.IncludedTraffic
			outgoingTraffic := server.OutgoingTraffic
			if includedTraffic > 0 {
				info.TrafficUsedPercent = math.Round(float64(outgoingTraffic) / float64(includedTraffic) * 100)
				trafficCache = info.TrafficUsedPercent
				lastTrafficUpdate = time.Now()
			}
		}
	}()

	wg.Wait()
	return info, nil
}

func main() {
	token := os.Getenv("HCLOUD_TOKEN")
	serverIDStr := os.Getenv("HCLOUD_SERVER_ID")
	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		log.Fatalf("Invalid server ID: %v", err)
	}

	client := hcloud.NewClient(hcloud.WithToken(token))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		info, err := getSystemInfo(client, serverID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(info); err != nil {
			log.Printf("Error while encoding: %v", err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
	})
	log.Println("Running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
