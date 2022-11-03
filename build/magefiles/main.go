//go:build mage

package main

import (
	"context"
	"os"
	"runtime"

	"dagger.io/dagger"
)

func ContainerWithDirectory(
	container *dagger.Container,
	path string,
	dir *dagger.Directory,
	opts ...dagger.DirectoryWithDirectoryOpts,
) *dagger.Container {
	return container.WithFS(container.FS().WithDirectory(path, dir, opts...))
}

func API(ctx context.Context) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	dir := client.Host().Workdir(dagger.HostWorkdirOpts{
		Exclude: []string{"dist"},
	})

	apiBinary := ContainerWithDirectory(client.
		Container().
		From("golang:latest"), "/src", dir).
		WithWorkdir("/src/projects/api").
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "mod", "download"},
		}).
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "build", "-o", "../../dist/api"},
		}).
		File("/src/dist/api")

	_, err = apiBinary.Export(ctx, "./dist/api")
	return err
}

func getOsArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
