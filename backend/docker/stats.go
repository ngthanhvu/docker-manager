package docker

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
)

func GetSystemInfo() (system.Info, error) {
	return Cli.Info(Ctx())
}

func GetVersion() (types.Version, error) {
	return Cli.ServerVersion(Ctx())
}

type DiskUsageSummary struct {
	TotalBytes int64 `json:"totalBytes"`
	UsedBytes  int64 `json:"usedBytes"`

	DockerUsedBytes  int64 `json:"dockerUsedBytes"`
	DockerTotalBytes int64 `json:"dockerTotalBytes"`

	WslUsedBytes  int64 `json:"wslUsedBytes"`
	WslTotalBytes int64 `json:"wslTotalBytes"`

	WindowsUsedBytes  int64 `json:"windowsUsedBytes"`
	WindowsTotalBytes int64 `json:"windowsTotalBytes"`
}

func GetDiskUsageSummary() (DiskUsageSummary, error) {

	du, err := Cli.DiskUsage(Ctx(), types.DiskUsageOptions{})
	if err != nil {
		return DiskUsageSummary{}, err
	}

	// ---------------------------
	// Calculate docker usage
	// ---------------------------

	var volumeBytes int64
	for _, v := range du.Volumes {
		if v.UsageData != nil {
			volumeBytes += v.UsageData.Size
		}
	}

	var imageBytes int64
	for _, img := range du.Images {
		imageBytes += img.Size
	}

	var containerBytes int64
	for _, c := range du.Containers {
		containerBytes += c.SizeRw
	}

	dockerObjectBytes := volumeBytes + imageBytes + containerBytes

	// ---------------------------
	// Get docker filesystem usage
	// ---------------------------

	info, err := Cli.Info(Ctx())

	var dockerTotalBytes int64
	var dockerFsUsedBytes int64

	if err == nil {

		dockerTotalBytes, dockerFsUsedBytes, err = getFSUsageBytes(info.DockerRootDir)

		if err != nil {
			dockerTotalBytes, dockerFsUsedBytes, _ = getFSUsageBytes("/")
		}

	}

	dockerUsedBytes := dockerObjectBytes
	if dockerUsedBytes <= 0 {
		dockerUsedBytes = dockerFsUsedBytes
	}

	if dockerTotalBytes > 0 && dockerUsedBytes > dockerTotalBytes {
		dockerUsedBytes = dockerTotalBytes
	}

	// ---------------------------
	// WSL filesystem
	// ---------------------------

	wslTotalBytes, wslUsedBytes, err := getFSUsageBytes("/")
	if err != nil {
		wslTotalBytes = 0
		wslUsedBytes = 0
	}

	// ---------------------------
	// Windows mount (C drive)
	// ---------------------------

	winTotalBytes, winUsedBytes, err := getFSUsageBytes("/mnt/c")
	if err != nil {
		winTotalBytes = 0
		winUsedBytes = 0
	}

	return DiskUsageSummary{
		TotalBytes: dockerTotalBytes,
		UsedBytes:  dockerUsedBytes,

		DockerUsedBytes:  dockerUsedBytes,
		DockerTotalBytes: dockerTotalBytes,

		WslUsedBytes:  wslUsedBytes,
		WslTotalBytes: wslTotalBytes,

		WindowsUsedBytes:  winUsedBytes,
		WindowsTotalBytes: winTotalBytes,
	}, nil
}

func getFSUsageBytes(path string) (int64, int64, error) {

	out, err := exec.Command("df", "-Pk", path).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("df failed for %s: %w", path, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("invalid df output")
	}

	fields := strings.Fields(lines[len(lines)-1])

	if len(fields) < 3 {
		return 0, 0, fmt.Errorf("cannot parse df output")
	}

	totalKB, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	usedKB, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return totalKB * 1024, usedKB * 1024, nil
}