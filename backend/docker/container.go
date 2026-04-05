package docker

import (
	"encoding/json"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

type ContainerResourceStats struct {
	CPUPercent       float64 `json:"cpuPercent"`
	MemoryUsedBytes  uint64  `json:"memoryUsedBytes"`
	MemoryLimitBytes uint64  `json:"memoryLimitBytes"`
	MemoryPercent    float64 `json:"memoryPercent"`
	NetworkRxBytes   uint64  `json:"networkRxBytes"`
	NetworkTxBytes   uint64  `json:"networkTxBytes"`
}

func ListContainers() ([]types.Container, error) {

	containers, err := Cli.ContainerList(
		Ctx(),
		container.ListOptions{All: true},
	)

	return containers, err
}

func StartContainer(id string) error {

	return Cli.ContainerStart(
		Ctx(),
		id,
		container.StartOptions{},
	)
}

func StopContainer(id string) error {

	return Cli.ContainerStop(
		Ctx(),
		id,
		container.StopOptions{},
	)
}

func RestartContainer(id string) error {

	return Cli.ContainerRestart(
		Ctx(),
		id,
		container.StopOptions{},
	)
}

func RemoveContainer(id string) error {

	return Cli.ContainerRemove(
		Ctx(),
		id,
		container.RemoveOptions{
			Force: true,
		},
	)
}

func PruneContainers() (any, error) {
	return Cli.ContainersPrune(
		Ctx(),
		filters.Args{},
	)
}

func ContainerStats(id string) (types.ContainerStats, error) {

	stats, err := Cli.ContainerStats(
		Ctx(),
		id,
		false,
	)

	return stats, err
}

func GetContainerResourceStats(id string) (ContainerResourceStats, error) {
	statsResp, err := Cli.ContainerStats(Ctx(), id, false)
	if err != nil {
		return ContainerResourceStats{}, err
	}
	defer statsResp.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		return ContainerResourceStats{}, err
	}

	result := ContainerResourceStats{
		CPUPercent:       calculateStatsCPUPercent(stats),
		MemoryUsedBytes:  calculateStatsMemoryUsage(stats),
		MemoryLimitBytes: stats.MemoryStats.Limit,
	}

	if result.MemoryLimitBytes > 0 {
		result.MemoryPercent = (float64(result.MemoryUsedBytes) / float64(result.MemoryLimitBytes)) * 100
		if result.MemoryPercent > 100 {
			result.MemoryPercent = 100
		}
	}

	for _, networkStat := range stats.Networks {
		result.NetworkRxBytes += networkStat.RxBytes
		result.NetworkTxBytes += networkStat.TxBytes
	}

	return result, nil
}

func GetBulkContainerResourceStats(ids []string) map[string]ContainerResourceStats {
	result := make(map[string]ContainerResourceStats, len(ids))
	for _, id := range ids {
		trimmedID := strings.TrimSpace(id)
		if trimmedID == "" {
			continue
		}
		stats, err := GetContainerResourceStats(trimmedID)
		if err != nil {
			continue
		}
		result[trimmedID] = stats
	}
	return result
}

func InspectContainer(id string) (types.ContainerJSON, error) {

	return Cli.ContainerInspect(
		Ctx(),
		id,
	)
}

func calculateStatsCPUPercent(stats types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}
	return (cpuDelta / systemDelta) * 100
}

func calculateStatsMemoryUsage(stats types.StatsJSON) uint64 {
	usage := stats.MemoryStats.Usage
	if usage == 0 {
		return 0
	}
	if cache, ok := stats.MemoryStats.Stats["cache"]; ok && usage > cache {
		return usage - cache
	}
	return usage
}
