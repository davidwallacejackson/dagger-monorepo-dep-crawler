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

func apiBinary(ctx context.Context, client *dagger.Client, goos string, goarch string) (*dagger.File, error) {

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
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	goos, goarch := getOsArch()
	binary, err := apiBinary(ctx, client, goos, goarch)
	if err != nil {
		return err
	}

	_, err = binary.Export(ctx, "./dist/api")
	return err
}

func frontendBuild(ctx context.Context, client *dagger.Client) (*dagger.Directory, error) {
	dir := client.Host().Workdir(dagger.HostWorkdirOpts{
		Exclude: []string{"dist", "projects/frontend/node_modules"},
	})

	dependencyScanner := depscanner.NewDependencyScanner(client, dir)

	sparseDir, err := dependencyScanner.GetSubdirWithDependencies(ctx, "projects/frontend")
	if err != nil {
		return nil, err
	}

	depsContainer := core.ContainerWithDirectory(client.
		Container().
		From("node:latest"), "/src", sparseDir).
		WithWorkdir("/src/projects/frontend").
		Exec(dagger.ContainerExecOpts{
			Args: []string{"yarn", "install", "--frozen-lockfile"},
		})

	return core.
		ContainerWithDirectory(depsContainer, "/src/projects/frontend", dir.Directory("projects/frontend")).
		WithEnvVariable("CI", "true").
		Exec(dagger.ContainerExecOpts{
			Args: []string{"yarn", "build"},
		}).
		Directory("/src/projects/frontend/dist"), nil
}

func Frontend(ctx context.Context) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	dist, err := frontendBuild(ctx, client)
	if err != nil {
		return err
	}

	_, err = dist.Export(ctx, "./dist/static")
	return err
}

func getOsArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}
