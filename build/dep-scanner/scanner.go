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
		Args: []string{"stat", "-c", "%F", relativePath},
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

	contents = strings.TrimSpace(contents)

	switch contents {
	case "directory":
		return "directory", nil
	case "regular file":
		return "file", nil
	case "regular empty file":
		return "file", nil
	default:
		return "", fmt.Errorf("unsupported path type: %s", contents)
	}
}
func (s *DependencyScanner) GetSubdirWithDependencies(ctx context.Context, relativePath string) (*dagger.Directory, error) {
	return s.getSubdirWithDependenciesInner(ctx, relativePath, true)
}

func (s *DependencyScanner) getSubdirWithDependenciesInner(ctx context.Context, relativePath string, sparse bool) (*dagger.Directory, error) {
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

	// just the dependencies that are directories
	var upstreamDirectories []string

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
			upstreamDirectories = append(upstreamDirectories, dependency)
		case "file":
			sparseDir = sparseDir.WithFile(dependency, s.ProjectRoot.File(dependency))
		default:
			fmt.Println("skipping", dependency)
		}
	}

	for _, upstreamDirectory := range upstreamDirectories {
		fmt.Println("upstream directory", upstreamDirectory)
		upstreamSparseDir, err := s.getSubdirWithDependenciesInner(ctx, upstreamDirectory, false)
		if err != nil {
			return nil, err
		}

		sparseDir = sparseDir.WithDirectory("/", upstreamSparseDir)
	}

	// if this is unsparse, we need the whole directory (not just whatever the scanner reported
	// that we need for resolving dependencies)
	if !sparse {
		sparseDir = sparseDir.WithDirectory(relativePath, s.ProjectRoot.Directory(relativePath))
	}

	return sparseDir, nil
}

func NewDependencyScanner(client *dagger.Client, projectRoot *dagger.Directory) DependencyScanner {
	return DependencyScanner{
		Client:      client,
		ProjectRoot: projectRoot,
	}
}
