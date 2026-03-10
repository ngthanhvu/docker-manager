package docker

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func ListNetworks() ([]types.NetworkResource, error) {
	return Cli.NetworkList(
		Ctx(),
		types.NetworkListOptions{},
	)
}

func RemoveNetwork(id string) error {
	return Cli.NetworkRemove(
		Ctx(),
		id,
	)
}

func PruneNetworks() (any, error) {
	return Cli.NetworksPrune(
		Ctx(),
		filters.Args{},
	)
}
