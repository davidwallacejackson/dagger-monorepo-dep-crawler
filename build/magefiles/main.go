//go:build mage

package main

import (
	"context"
	"os"
	"runtime"

	"dagger.io/dagger"
	depscanner "github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/core"
)

func apiBinary(ctx context.Context, goos string, goarch string) (*dagger.File, error) {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return nil, err
	}

	defer client.Close()

	dir := client.Host().Workdir(dagger.HostWorkdirOpts{
		Exclude: []string{"dist"},
	})

	dependencyScanner := depscanner.NewDependencyScanner(client, dir)

	sparseDir, err := dependencyScanner.GetSubdirWithDependencies(ctx, "projects/api")
	if err != nil {
		return nil, err
	}

	depsContainer := core.ContainerWithDirectory(client.
		Container().
		From("golang:latest"), "/src", sparseDir).
		WithWorkdir("/src/projects/api").
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "mod", "download"},
		})

	return core.
		ContainerWithDirectory(depsContainer, "/src/projects/api", dir.Directory("projects/api")).
		WithEnvVariable("GOOS", goos).
		WithEnvVariable("GOARCH", goarch).
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "build", "-o", "../../dist/api"},
		}).
		File("/src/dist/api"), nil
}

func APIDev(ctx context.Context) error {
	goos, goarch := getOsArch()
	binary, err := apiBinary(ctx, goos, goarch)
	if err != nil {
		return err
	}

	_, err = binary.Export(ctx, "./dist/api")
	return err
}

func getOsArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
