//go:build mage

package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"dagger.io/dagger"
	depscanner "github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/core"
)

func getOsArch() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

func getWorkdir(client *dagger.Client) *dagger.Directory {
	return client.Host().Workdir(dagger.HostWorkdirOpts{
		Exclude: []string{"dist", "projects/frontend/node_modules"},
	})
}

func goBuilder(ctx context.Context, client *dagger.Client, projectDir string, goos string, goarch string) (*dagger.Container, error) {
	dir := getWorkdir(client)
	dependencyScanner := depscanner.NewDependencyScanner(client, dir)

	projectPath := filepath.Join("projects", projectDir)

	sparseDir, err := dependencyScanner.GetSubdirWithDependencies(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	depsContainer := core.ContainerWithDirectory(client.
		Container().
		From("golang:latest"), "/src", sparseDir).
		WithWorkdir(filepath.Join("/src", projectPath)).
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "mod", "download"},
		})

	return core.
		ContainerWithDirectory(depsContainer, filepath.Join("/src", projectPath), dir.Directory(projectPath)).
		WithEnvVariable("GOOS", goos).
		WithEnvVariable("GOARCH", goarch).
		WithEnvVariable("CGO_ENABLED", "0"), nil
}

func apiBinary(ctx context.Context, client *dagger.Client, goos string, goarch string) (*dagger.File, error) {
	container, err := goBuilder(ctx, client, "api", goos, goarch)
	if err != nil {
		return nil, err
	}

	return container.
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

func apiProd(ctx context.Context, client *dagger.Client) (*dagger.File, error) {
	_, goarch := getOsArch()
	return apiBinary(ctx, client, "linux", goarch)
}

func frontendBuild(ctx context.Context, client *dagger.Client) (*dagger.Directory, error) {
	dir := getWorkdir(client)
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

func DockerContainer(ctx context.Context) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	api, err := apiProd(ctx, client)
	if err != nil {
		return err
	}

	frontend, err := frontendBuild(ctx, client)
	if err != nil {
		return err
	}

	withApi := core.ContainerWithFile(client.Container().From("alpine:latest"), "/app/api", api)
	withFrontend := core.ContainerWithDirectory(withApi, "/app/static", frontend)

	final := withFrontend.
		WithWorkdir("/app").
		WithEntrypoint([]string{"/app/api"})

	_, err = final.Export(ctx, "./dist/container.tgz")
	return err
}

func Cli(ctx context.Context) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}

	defer client.Close()

	goos, goarch := getOsArch()
	builder, err := goBuilder(ctx, client, "cli", goos, goarch)
	if err != nil {
		return err
	}

	cli := builder.
		Exec(dagger.ContainerExecOpts{
			Args: []string{"go", "build", "-o", "../../dist/cli"},
		}).
		File("/src/dist/cli")

	_, err = cli.Export(ctx, "./dist/cli")
	return err
}
