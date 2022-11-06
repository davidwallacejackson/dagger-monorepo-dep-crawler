package node

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"dagger.io/dagger"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/core"
	"github.com/rs/zerolog"
)

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func NodeStrategy(ctx context.Context, logger zerolog.Logger, projectRoot *dagger.Directory, relativePath string) ([]string, error) {
	var dependencies = []string{}

	dir := projectRoot.Directory(relativePath)

	logger.Trace().Msg("Checking for package.json file")
	packageJsonFile, err := dir.File("package.json").Contents(ctx)
	if err != nil {
		return nil, nil
	}
	logger.Trace().Msg("Found package.json file")
	dependencies = append(dependencies, core.ResolveRelativePath(relativePath, "package.json"))

	if core.FileExists(ctx, dir.File("package-lock.json")) {
		dependencies = append(dependencies, core.ResolveRelativePath(relativePath, "package-lock.json"))
	}

	if core.FileExists(ctx, dir.File("yarn.lock")) {
		dependencies = append(dependencies, core.ResolveRelativePath(relativePath, "yarn.lock"))
	}

	if core.FileExists(ctx, dir.File("pnpm-lock.yaml")) {
		dependencies = append(dependencies, core.ResolveRelativePath(relativePath, "pnpm-lock.yaml"))
	}

	var parsedPackageJson PackageJSON

	err = json.Unmarshal([]byte(packageJsonFile), &parsedPackageJson)
	if err != nil {
		logger.Trace().Err(err).Msg("Failed to parse package.json file")
		return dependencies, nil
	}

	allDeps := make(map[string]string)

	for depName, depVersion := range parsedPackageJson.Dependencies {
		allDeps[depName] = depVersion
	}

	for depName, depVersion := range parsedPackageJson.DevDependencies {
		if _, ok := allDeps[depName]; ok {
			logger.Warn().Msgf("Dependency %s (version %s) is overridden by devDependency (version %s)", depName, allDeps[depName], depVersion)
		}
		allDeps[depName] = depVersion
	}

	pathRegexp := regexp.MustCompile(`(?:file|link):(.*)$`)

	for depName, depVersion := range allDeps {
		// TODO: npm and pnpm formats
		if strings.HasPrefix(depVersion, "link:") || strings.HasPrefix(depVersion, "file:") {
			matches := pathRegexp.FindStringSubmatch(depVersion)
			if len(matches) != 2 {
				logger.Warn().Msgf("Failed to parse path from dependency %s (version %s)", depName, depVersion)
				continue
			}
			depPath := matches[1]
			resolvedDepPath := core.ResolveRelativePath(relativePath, depPath)

			if !core.DirectoryExists(ctx, projectRoot.Directory(resolvedDepPath)) {
				logger.Warn().Msgf("Dependency %s (version %s) points to non-existent directory %s", depName, depVersion, resolvedDepPath)
				continue
			}
			dependencies = append(dependencies, resolvedDepPath)
		}
	}
	return dependencies, nil
}
