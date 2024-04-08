package step

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/cache"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	xcodecache "github.com/bitrise-io/go-xcode/v2/xcodecache"
)

const (
	stepId = "save-spm-cache"

	// Cache key template
	// OS + Arch: SPM works on Linux too, and Intel/ARM difference is important on macOS
	// checksum: Package.resolved is the dependency lockfile, either in the project root (pure Swift project)
	// or at project.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved
	key = `{{ .OS }}-{{ .Arch }}-spm-cache-{{ checksum "**/Package.resolved" }}`
)

type Input struct {
	Verbose         bool   `env:"verbose,required"`
	DerivedDataPath string `env:"derived_data_path"`
	ProjectPath     string `env:"project_path"`
}

type SaveCacheStep struct {
	logger                  log.Logger
	inputParser             stepconf.InputParser
	pathChecker             pathutil.PathChecker
	pathProvider            pathutil.PathProvider
	pathModifier            pathutil.PathModifier
	envRepo                 env.Repository
	derivedDataPathProvider xcodecache.SwiftPackageCache
}

func New(
	logger log.Logger,
	inputParser stepconf.InputParser,
	pathChecker pathutil.PathChecker,
	pathProvider pathutil.PathProvider,
	pathModifier pathutil.PathModifier,
	envRepo env.Repository,
	derivedDataPathProvider xcodecache.SwiftPackageCache,
) SaveCacheStep {
	return SaveCacheStep{
		logger:                  logger,
		inputParser:             inputParser,
		pathChecker:             pathChecker,
		pathProvider:            pathProvider,
		pathModifier:            pathModifier,
		envRepo:                 envRepo,
		derivedDataPathProvider: derivedDataPathProvider,
	}
}

func (step SaveCacheStep) Run() error {
	var input Input
	if err := step.inputParser.Parse(&input); err != nil {
		return fmt.Errorf("failed to parse inputs: %w", err)
	}
	stepconf.Print(input)

	input.DerivedDataPath = strings.TrimSpace(input.DerivedDataPath)
	input.ProjectPath = strings.TrimSpace(input.ProjectPath)
	if input.DerivedDataPath == "" && input.ProjectPath == "" {
		return fmt.Errorf("failed to parse inputs: provide either Derived Data Path (derived_data_path) or Xcode Project Path (project_path) Inputs")
	}
	if input.DerivedDataPath != "" && input.ProjectPath != "" {
		step.logger.Warnf("Both Derived Data Path (derived_data_path) and Xcode Project Path (project_path) Inputs are provided, only derived_data_path is used, project_path is ignored")
	}

	path := filepath.Join(input.DerivedDataPath, "SourcePackages")
	if input.ProjectPath != "" {
		var err error
		// project specific path already contains SourcePacages ($HOME/Library/Developer/Xcode/DerivedData/[PER_PROJECT_DERIVED_DATA]/SourcePackages)
		if path, err = step.derivedDataPathProvider.SwiftPackagesPath(input.ProjectPath); err != nil {
			return fmt.Errorf("failed to get Derived Data Path: %w", err)
		}
	}

	step.logger.Println()
	step.logger.Printf("Cache key: %s", key)
	step.logger.Printf("Cache paths:")
	step.logger.Printf(path)
	step.logger.Println()

	step.logger.EnableDebugLog(input.Verbose)

	saver := cache.NewSaver(step.envRepo, step.logger, step.pathProvider, step.pathModifier, step.pathChecker)
	return saver.Save(cache.SaveCacheInput{
		StepId:      stepId,
		Verbose:     input.Verbose,
		Key:         key,
		Paths:       []string{path},
		IsKeyUnique: true,
	})
}
