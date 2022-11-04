package depscanner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/core"
)

var strategies = map[string]core.ScanStrategy{}

func RegisterStrategy(name string, strategy core.ScanStrategy) {
	strategies[name] = strategy
}

func init() {
	RegisterStrategy("depends-file", core.DependsFileStrategy)
}

type DependencyScanner struct {
	Client      *dagger.Client
	ProjectRoot *dagger.Directory
}

func (s *DependencyScanner) pathType(ctx context.Context, dir *dagger.Directory, relativePath string) (string, error) {
	withDir := core.ContainerWithDirectory(s.Client.Container().
		From("alpine:latest"), "/src", dir).
		WithWorkdir("/src")

	output := withDir.Exec(dagger.ContainerExecOpts{
		Args: []string{"ls", "-l", relativePath},
	})

	exitCode, err := output.ExitCode(ctx)
	if err != nil {
		return "", err
	}

	if exitCode != 0 {
		return "invalid", nil
	}

	contents, err := output.Stdout().Contents(ctx)
	if err != nil {
		return "", err
	}

	if contents[0] == 'd' {
		return "directory", nil
	}

	return "file", nil
}

func (s *DependencyScanner) GetSubdirWithDependencies(ctx context.Context, relativePath string) (*dagger.Directory, error) {
	var dependencies []string

	// collect a list of file/directory dependencies as paths relative to ProjectRoot
	for _, strategy := range strategies {
		strategyDependencies, err := strategy(ctx, s.ProjectRoot, relativePath)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, strategyDependencies...)
	}

	var cleanedDependencies []string

	for _, dependency := range dependencies {
		trimmed := strings.TrimSpace(dependency)

		if trimmed == "" {
			continue
		}

		cleaned := filepath.Clean(filepath.Join(relativePath, trimmed))

		cleanedDependencies = append(cleanedDependencies, cleaned)
	}

	fmt.Printf("Cleaned dependencies: %v\n", cleanedDependencies)

	sparseDir := s.Client.Directory()

	for _, dependency := range cleanedDependencies {
		pathType, err := s.pathType(ctx, s.ProjectRoot, dependency)
		if err != nil {
			return nil, err
		}

		fmt.Println("dependency", dependency, "is a", pathType)

		switch pathType {
		case "directory":
			sparseDir = sparseDir.WithDirectory(dependency, s.ProjectRoot.Directory(dependency))
		case "file":
			sparseDir = sparseDir.WithFile(dependency, s.ProjectRoot.File(dependency))
		default:
			fmt.Println("skipping", dependency)
		}
	}

	return sparseDir, nil
}

func NewDependencyScanner(client *dagger.Client, projectRoot *dagger.Directory) DependencyScanner {
	return DependencyScanner{
		Client:      client,
		ProjectRoot: projectRoot,
	}
}
