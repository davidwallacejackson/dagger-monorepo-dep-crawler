package depscanner

import (
	"context"
	"fmt"

	"dagger.io/dagger"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/core"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/golang"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/node"
	"github.com/fatih/color"
	"github.com/rs/zerolog"
)

var strategies = map[string]StrategyConfig{}
var logger zerolog.Logger

func RegisterStrategy(name string, strategy core.ScanStrategy) {
	strategies[name] = StrategyConfig{
		Name: name,
		Func: strategy,
	}
}

type StrategyConfig struct {
	Name string
	Func core.ScanStrategy
}

func init() {
	RegisterStrategy("go", golang.GoModStrategy)
	RegisterStrategy("node", node.NodeStrategy)
	RegisterStrategy("depends-file", core.DependsFileStrategy)

	writer := zerolog.NewConsoleWriter()
	writer.FieldsExclude = []string{"strategy"}
	writer.PartsOrder = []string{
		zerolog.LevelFieldName,
		"strategy",
		zerolog.MessageFieldName,
	}

	logger = zerolog.New(writer)
}

type DependencyScanner struct {
	Client      *dagger.Client
	ProjectRoot *dagger.Directory
}

func (s *DependencyScanner) pathType(ctx context.Context, dir *dagger.Directory, relativePath string) (string, error) {
	_, err := dir.File(relativePath).Contents(ctx)
	if err == nil {
		return "file", nil
	}

	_, err = dir.Directory(relativePath).Entries(ctx)
	if err == nil {
		return "directory", nil
	}

	return "", fmt.Errorf("path %s is neither a file nor a directory", relativePath)
}
func (s *DependencyScanner) GetSubdirWithDependencies(ctx context.Context, relativePath string) (*dagger.Directory, error) {
	return s.getSubdirWithDependenciesInner(ctx, relativePath, true)
}

func (s *DependencyScanner) getSubdirWithDependenciesInner(ctx context.Context, relativePath string, sparse bool) (*dagger.Directory, error) {
	var dependencies []string

	// collect a list of file/directory dependencies as paths relative to ProjectRoot
	for _, strategy := range strategies {
		strategyLogger := logger.With().Str("strategy", color.HiCyanString(strategy.Name)).Logger()
		strategyDependencies, err := strategy.Func(ctx, strategyLogger, s.ProjectRoot, relativePath)
		if err != nil {
			return nil, err
		}
		dependencies = append(dependencies, strategyDependencies...)
	}

	// just the dependencies that are directories
	var upstreamDirectoryPaths []string

	fmt.Printf("Cleaned dependencies: %v\n", dependencies)

	output := s.Client.Directory()

	for _, dependency := range dependencies {
		pathType, err := s.pathType(ctx, s.ProjectRoot, dependency)
		if err != nil {
			return nil, err
		}

		fmt.Println("dependency", dependency, "is a", pathType)

		switch pathType {
		case "directory":
			output = output.WithDirectory(dependency, s.ProjectRoot.Directory(dependency))
			upstreamDirectoryPaths = append(upstreamDirectoryPaths, dependency)
		case "file":
			output = output.WithFile(dependency, s.ProjectRoot.File(dependency))
		default:
			fmt.Println("skipping", dependency)
		}
	}

	for _, upstreamDirectoryPath := range upstreamDirectoryPaths {
		fmt.Println("upstream directory", upstreamDirectoryPath)
		upstreamDir, err := s.getSubdirWithDependenciesInner(ctx, upstreamDirectoryPath, false)
		if err != nil {
			return nil, err
		}

		output = output.WithDirectory("/", upstreamDir)
	}

	// if this is unsparse, we need the whole directory (not just whatever the scanner reported
	// that we need for resolving dependencies)
	if !sparse {
		output = output.WithDirectory(relativePath, s.ProjectRoot.Directory(relativePath))
	}

	return output, nil
}

func NewDependencyScanner(client *dagger.Client, projectRoot *dagger.Directory) DependencyScanner {
	return DependencyScanner{
		Client:      client,
		ProjectRoot: projectRoot,
	}
}
