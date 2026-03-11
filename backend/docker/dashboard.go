package docker

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	containerapi "github.com/docker/docker/api/types/container"
)

type DashboardNetworkMetric struct {
	RxBytes     uint64  `json:"rxBytes"`
	TxBytes     uint64  `json:"txBytes"`
	RxRateBytes float64 `json:"rxRateBytes"`
	TxRateBytes float64 `json:"txRateBytes"`
}

type DashboardDiskMetric struct {
	ReadBytes      uint64  `json:"readBytes"`
	WriteBytes     uint64  `json:"writeBytes"`
	ReadRateBytes  float64 `json:"readRateBytes"`
	WriteRateBytes float64 `json:"writeRateBytes"`
}

type DashboardMetricPoint struct {
	Timestamp       time.Time                         `json:"timestamp"`
	CPUPercent      float64                           `json:"cpuPercent"`
	MemoryPercent   float64                           `json:"memoryPercent"`
	MemoryUsedBytes uint64                            `json:"memoryUsedBytes"`
	MemoryTotalBytes uint64                           `json:"memoryTotalBytes"`
	Networks        map[string]DashboardNetworkMetric `json:"networks"`
	Disk            DashboardDiskMetric               `json:"disk"`
}

type DashboardMetricsResponse struct {
	Points     []DashboardMetricPoint `json:"points"`
	Interfaces []string               `json:"interfaces"`
}

var dashboardMetricsState = struct {
	mu      sync.Mutex
	previous *DashboardMetricPoint
	history []DashboardMetricPoint
}{}

func GetDashboardMetrics(limit int) (DashboardMetricsResponse, error) {
	if limit <= 0 {
		limit = 36
	}
	if limit > 180 {
		limit = 180
	}

	point, err := collectDashboardMetricPoint()
	if err != nil {
		return DashboardMetricsResponse{}, err
	}

	dashboardMetricsState.mu.Lock()
	defer dashboardMetricsState.mu.Unlock()

	applyMetricRates(dashboardMetricsState.previous, &point)
	copiedPoint := cloneDashboardMetricPoint(point)
	dashboardMetricsState.previous = &copiedPoint
	dashboardMetricsState.history = append(dashboardMetricsState.history, copiedPoint)
	if len(dashboardMetricsState.history) > limit {
		dashboardMetricsState.history = append([]DashboardMetricPoint(nil), dashboardMetricsState.history[len(dashboardMetricsState.history)-limit:]...)
	}

	points := make([]DashboardMetricPoint, 0, len(dashboardMetricsState.history))
	interfaceSet := make(map[string]struct{})
	for _, item := range dashboardMetricsState.history {
		cloned := cloneDashboardMetricPoint(item)
		points = append(points, cloned)
		for name := range cloned.Networks {
			interfaceSet[name] = struct{}{}
		}
	}

	interfaces := make([]string, 0, len(interfaceSet))
	for name := range interfaceSet {
		interfaces = append(interfaces, name)
	}
	sort.Strings(interfaces)

	return DashboardMetricsResponse{
		Points:     points,
		Interfaces: interfaces,
	}, nil
}

func collectDashboardMetricPoint() (DashboardMetricPoint, error) {
	info, err := Cli.Info(Ctx())
	if err != nil {
		return DashboardMetricPoint{}, err
	}

	containers, err := Cli.ContainerList(Ctx(), containerapi.ListOptions{})
	if err != nil {
		return DashboardMetricPoint{}, err
	}

	point := DashboardMetricPoint{
		Timestamp:        time.Now().UTC(),
		MemoryTotalBytes: uint64(info.MemTotal),
		Networks:         make(map[string]DashboardNetworkMetric),
	}

	for _, container := range containers {
		statsResp, err := Cli.ContainerStats(Ctx(), container.ID, false)
		if err != nil {
			continue
		}

		var stats types.StatsJSON
		decodeErr := json.NewDecoder(statsResp.Body).Decode(&stats)
		statsResp.Body.Close()
		if decodeErr != nil {
			continue
		}

		point.CPUPercent += calculateContainerCPUPercent(stats)
		point.MemoryUsedBytes += calculateContainerMemoryUsage(stats)
		accumulateNetworkMetrics(point.Networks, stats.Networks)
		readBytes, writeBytes := collectDiskBytes(stats.BlkioStats.IoServiceBytesRecursive)
		point.Disk.ReadBytes += readBytes
		point.Disk.WriteBytes += writeBytes
	}

	if point.MemoryTotalBytes > 0 {
		point.MemoryPercent = (float64(point.MemoryUsedBytes) / float64(point.MemoryTotalBytes)) * 100
	}
	if point.CPUPercent > 100 {
		point.CPUPercent = 100
	}
	if point.MemoryPercent > 100 {
		point.MemoryPercent = 100
	}

	return point, nil
}

func calculateContainerCPUPercent(stats types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}
	return (cpuDelta / systemDelta) * 100
}

func calculateContainerMemoryUsage(stats types.StatsJSON) uint64 {
	usage := stats.MemoryStats.Usage
	if usage == 0 {
		return 0
	}
	if cache, ok := stats.MemoryStats.Stats["cache"]; ok && usage > cache {
		return usage - cache
	}
	return usage
}

func accumulateNetworkMetrics(target map[string]DashboardNetworkMetric, stats map[string]types.NetworkStats) {
	for name, networkStat := range stats {
		current := target[name]
		current.RxBytes += networkStat.RxBytes
		current.TxBytes += networkStat.TxBytes
		target[name] = current
	}
}

func collectDiskBytes(entries []types.BlkioStatEntry) (uint64, uint64) {
	var readBytes uint64
	var writeBytes uint64

	for _, entry := range entries {
		switch strings.ToLower(entry.Op) {
		case "read":
			readBytes += entry.Value
		case "write":
			writeBytes += entry.Value
		}
	}

	return readBytes, writeBytes
}

func applyMetricRates(previous *DashboardMetricPoint, current *DashboardMetricPoint) {
	if previous == nil {
		return
	}

	deltaSeconds := current.Timestamp.Sub(previous.Timestamp).Seconds()
	if deltaSeconds <= 0 {
		return
	}

	for name, currentNet := range current.Networks {
		previousNet := previous.Networks[name]
		if currentNet.RxBytes >= previousNet.RxBytes {
			currentNet.RxRateBytes = float64(currentNet.RxBytes-previousNet.RxBytes) / deltaSeconds
		}
		if currentNet.TxBytes >= previousNet.TxBytes {
			currentNet.TxRateBytes = float64(currentNet.TxBytes-previousNet.TxBytes) / deltaSeconds
		}
		current.Networks[name] = currentNet
	}

	if current.Disk.ReadBytes >= previous.Disk.ReadBytes {
		current.Disk.ReadRateBytes = float64(current.Disk.ReadBytes-previous.Disk.ReadBytes) / deltaSeconds
	}
	if current.Disk.WriteBytes >= previous.Disk.WriteBytes {
		current.Disk.WriteRateBytes = float64(current.Disk.WriteBytes-previous.Disk.WriteBytes) / deltaSeconds
	}
}

func cloneDashboardMetricPoint(point DashboardMetricPoint) DashboardMetricPoint {
	clonedNetworks := make(map[string]DashboardNetworkMetric, len(point.Networks))
	for name, metric := range point.Networks {
		clonedNetworks[name] = metric
	}

	return DashboardMetricPoint{
		Timestamp:        point.Timestamp,
		CPUPercent:       point.CPUPercent,
		MemoryPercent:    point.MemoryPercent,
		MemoryUsedBytes:  point.MemoryUsedBytes,
		MemoryTotalBytes: point.MemoryTotalBytes,
		Networks:         clonedNetworks,
		Disk:             point.Disk,
	}
}
