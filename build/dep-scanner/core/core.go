package core

import (
	"context"
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

	cleanedLines := []string{".depends-on"}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	return cleanedLines, nil
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
