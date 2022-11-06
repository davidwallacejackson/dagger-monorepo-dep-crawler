package core

import (
	"context"
	"path/filepath"
	"strings"

	"dagger.io/dagger"

	"github.com/rs/zerolog"
)

type ScanStrategy func(ctx context.Context, logger zerolog.Logger, projectRoot *dagger.Directory, relativePath string) ([]string, error)

const dependsFileName = ".depends-on"

func DependsFileStrategy(ctx context.Context, logger zerolog.Logger, projectRoot *dagger.Directory, relativePath string) ([]string, error) {
	logger.Trace().Msg("Checking for depends file")
	dir := projectRoot.Directory(relativePath)

	if !FileExists(ctx, dir.File(dependsFileName)) {
		logger.Trace().Msg("No depends file found")
		return nil, nil
	}

	dependencies := []string{ResolveRelativePath(relativePath, dependsFileName)}

	dependsFile, err := dir.File(dependsFileName).Contents(ctx)
	if err != nil {
		return nil, err
	}

	logger.Trace().Msgf("Found depends file")
	lines := strings.Split(string(dependsFile), "\n")

	for _, dependency := range lines {
		dependency = strings.TrimSpace(dependency)
		if dependency == "" {
			continue
		}
		dependencies = append(dependencies, ResolveRelativePath(relativePath, dependency))
	}

	return dependencies, nil
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

func FileExists(ctx context.Context, file *dagger.File) bool {
	_, err := file.Contents(ctx)
	return err == nil
}

func DirectoryExists(ctx context.Context, dir *dagger.Directory) bool {
	_, err := dir.Entries(ctx)
	return err == nil
}
