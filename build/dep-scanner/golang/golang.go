package golang

import (
	"context"
	"fmt"
	"path/filepath"

	"dagger.io/dagger"
	"github.com/davidwallacejackson/dagger-monorepo-dep-crawler/build/dep-scanner/core"
	"github.com/rs/zerolog"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func findGoWork(ctx context.Context, projectRoot *dagger.Directory, searchPath string) (*string, error) {
	workPath := filepath.Join(searchPath, "go.work")
	f := projectRoot.File(workPath)

	exists := core.FileExists(ctx, f)
	if !exists {
		nextSearchPath := filepath.Dir(searchPath)

		if nextSearchPath == searchPath {
			return nil, nil
		}

		return findGoWork(ctx, projectRoot, nextSearchPath)
	}

	return &workPath, nil
}

func getModuleSpec(version module.Version) string {
	return fmt.Sprintf("%s@%s", version.Path, version.Version)
}

func GoModStrategy(ctx context.Context, logger zerolog.Logger, projectRoot *dagger.Directory, relativePath string) ([]string, error) {
	var dependencies = []string{}

	dir := projectRoot.Directory(relativePath)

	goModFile, err := dir.File("go.mod").Contents(ctx)
	if err != nil {
		return nil, nil
	}
	dependencies = append(dependencies, core.ResolveRelativePath(relativePath, "go.mod"))

	goModParsed, err := modfile.Parse("go.mod", []byte(goModFile), nil)
	if err != nil {
		// if the modfile is unparseable, it should still be considered a dependency
		// but there's no reason to add the rest
		return dependencies, nil
	}

	if core.FileExists(ctx, dir.File("go.sum")) {
		dependencies = append(dependencies, core.ResolveRelativePath(relativePath, "go.sum"))
	}

	// we build replace directives from go.mod first, then override with go.work
	// if it's present
	//
	// note that we don't actually add the dependencies here, since we haven't
	// validated that they are actually required
	var moduleReplacements = map[string]string{}

	for _, replace := range goModParsed.Replace {
		hypotheticalDirPath := core.ResolveRelativePath(relativePath, replace.New.Path)
		dirExists := core.DirectoryExists(ctx, projectRoot.Directory(hypotheticalDirPath))

		if dirExists {
			moduleReplacements[getModuleSpec(replace.Old)] = hypotheticalDirPath
		}
	}

	// TODO: use the parsed go.work file to override replacements from go.mod

	// workPath, err := findGoWork(ctx, projectRoot, relativePath)
	// if err != nil {
	// 	return nil, nil
	// }

	// if workPath != nil {
	// 	workFile, err := dir.File(*workPath).Contents(ctx)
	// 	if err == nil {
	// 		workParsed, err := modfile.ParseWork(*workPath, []byte(workFile), nil)
	// 		if err == nil {
	// 			return nil, nil
	// 		}

	// 		var deps []string
	// 		for _, require := range workParsed.Replace {
	// 			deps = append(deps, require.Mod.Path)
	// 		}

	// 	}

	// 	return deps, nil
	// }

	for _, require := range goModParsed.Require {
		modulePath, ok := moduleReplacements[getModuleSpec(require.Mod)]

		if !ok {
			// only modules replaced with filesystem paths are dependencies
			continue
		}

		dependencies = append(dependencies, modulePath)
	}

	return dependencies, nil
}

var _ core.ScanStrategy = GoModStrategy
