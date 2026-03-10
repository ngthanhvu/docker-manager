package docker

import (
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
)

func ListVolumes() ([]*volume.Volume, error) {
	resp, err := Cli.VolumeList(
		Ctx(),
		volume.ListOptions{},
	)
	if err != nil {
		return nil, err
	}
	return resp.Volumes, nil
}

func RemoveVolume(id string) error {
	return Cli.VolumeRemove(
		Ctx(),
		id,
		true,
	)
}

func PruneVolumes() (any, error) {
	return Cli.VolumesPrune(
		Ctx(),
		filters.Args{},
	)
}
