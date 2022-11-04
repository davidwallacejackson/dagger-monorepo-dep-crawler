package core

import (
	"context"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

type ScanStrategy = func(ctx context.Context, projectRoot *dagger.Directory, relativePath string) ([]string, error)

func DependsFileStrategy(ctx context.Context, projectRoot *dagger.Directory, relativePath string) ([]string, error) {
	dir := projectRoot.Directory(relativePath)

	dependsFile, err := dir.File(".depends-on").Contents(ctx)
	if err != nil {
		// TODO: assuming for the moment that the only possible error
		// is that the file doesn't exist. If we can catch something
		// more specific that'd be better.
		return nil, nil
	}

	lines := strings.Split(string(dependsFile), "\n")

	cleanedDependencies := []string{".depends-on"}

	for _, dependency := range lines {
		dependency = strings.TrimSpace(dependency)
		if dependency == "" {
			continue
		}
		cleanedDependencies = append(cleanedDependencies, ResolveRelativePath(relativePath, dependency))
	}

	return cleanedDependencies, nil
}

var _ ScanStrategy = DependsFileStrategy

func ContainerWithDirectory(
	container *dagger.Container,
	path string,
	dir *dagger.Directory,
	opts ...dagger.DirectoryWithDirectoryOpts,
) *dagger.Container {
	return container.WithFS(container.FS().WithDirectory(path, dir, opts...))
}

func ContainerWithFile(
	container *dagger.Container,
	path string,
	file *dagger.File,
) *dagger.Container {
	return container.WithFS(container.FS().WithFile(path, file))
}

func ResolveRelativePath(
	basePath string,
	relativePath string,
) string {
	basePath = strings.TrimSpace(basePath)
	relativePath = strings.TrimSpace(relativePath)

	return filepath.Clean(filepath.Join(basePath, relativePath))
}
