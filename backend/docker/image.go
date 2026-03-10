package docker

import (
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

func ListImages() ([]image.Summary, error) {
	return Cli.ImageList(
		Ctx(),
		image.ListOptions{All: true},
	)
}

func RemoveImage(id string) error {
	_, err := Cli.ImageRemove(
		Ctx(),
		id,
		image.RemoveOptions{Force: true},
	)
	return err
}

func PruneImages() (any, error) {
	return Cli.ImagesPrune(
		Ctx(),
		filters.NewArgs(filters.Arg("dangling", "false")),
	)
}
