package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

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

func InspectContainer(id string) (types.ContainerJSON, error) {

	return Cli.ContainerInspect(
		Ctx(),
		id,
	)
}
