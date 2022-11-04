//go:build mage

package main

import (
	"context"

	"dagger.io/dagger"
	depscanner "github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/core"
)

func API(ctx context.Context) error {
	client, err := dagger.Connect(ctx)
	if err != nil {
		return err
	}

	defer client.Close()

	dir := client.Host().Workdir(dagger.HostWorkdirOpts{
		Exclude: []string{"dist"},
	})

	dependencyScanner := depscanner.NewDependencyScanner(client, dir)

	sparseDir, err := dependencyScanner.GetSubdirWithDependencies(ctx, "projects/api")
	if err != nil {
		return err
	}

	depsContainer := core.ContainerWithDirectory(client.
		Container().
		From("golang:latest"), "/src", sparseDir).
		WithWorkdir("/src/projects/api").
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "mod", "download"},
		})

	apiBinary := core.ContainerWithDirectory(depsContainer, "/src/projects/api", dir.Directory("projects/api")).
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "build", "-o", "../../dist/api"},
		}).
		File("/src/dist/api")

	_, err = apiBinary.Export(ctx, "./dist/api")
	return err
}
