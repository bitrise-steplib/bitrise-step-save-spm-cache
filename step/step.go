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
	stepID = "save-spm-cache"

	// Cache key template
	// OS + Arch: SPM works on Linux too, and Intel/ARM difference is important on macOS
	// checksum: Package.resolved is the dependency lockfile, either in the project root (pure Swift project)
	// or at project.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved
	key = `{{ .OS }}-{{ .Arch }}-spm-cache-{{ checksum "**/Package.resolved" }}`
)

type Input struct {
	Verbose          bool   `env:"verbose,required"`
	DerivedDataPath  string `env:"derived_data_path"`
	ProjectPath      string `env:"project_path"`
	CompressionLevel int    `env:"compression_level,range[1..19]"`
}

type Config struct {
	CachePaths       string
	IsVerbose        bool
	CompressionLevel int
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

func (step SaveCacheStep) ProcessConfig() (Config, error) {
	var input Input
	if err := step.inputParser.Parse(&input); err != nil {
		return Config{}, err
	}
	stepconf.Print(input)
	step.logger.EnableDebugLog(input.Verbose)

	input.DerivedDataPath = strings.TrimSpace(input.DerivedDataPath)
	input.ProjectPath = strings.TrimSpace(input.ProjectPath)
	if input.DerivedDataPath == "" && input.ProjectPath == "" {
		return Config{}, fmt.Errorf("provide either Derived Data Path (derived_data_path) or Xcode Project Path (project_path) Inputs")
	}
	if input.DerivedDataPath != "" && input.ProjectPath != "" {
		input.ProjectPath = ""
		step.logger.Warnf("Both Derived Data Path (derived_data_path) and Xcode Project Path (project_path) Inputs are provided, only derived_data_path is used, project_path is ignored")
	}

	sourcePackagesPath := filepath.Join(input.DerivedDataPath, "SourcePackages")
	if input.ProjectPath != "" {
		var err error
		if input.ProjectPath, err = step.pathModifier.AbsPath(input.ProjectPath); err != nil {
			return Config{}, fmt.Errorf("failed to expand project path: %w", err)
		}
		// project specific path already contains SourcePacages ($HOME/Library/Developer/Xcode/DerivedData/[PER_PROJECT_DERIVED_DATA]/SourcePackages)
		if sourcePackagesPath, err = step.derivedDataPathProvider.SwiftPackagesPath(input.ProjectPath); err != nil {
			return Config{}, fmt.Errorf("failed to get Derived Data Path: %w", err)
		}
	}

	return Config{
		CachePaths:       sourcePackagesPath,
		IsVerbose:        input.Verbose,
		CompressionLevel: input.CompressionLevel,
	}, nil
}

func (step SaveCacheStep) Run(config Config) error {
	step.logger.Println()
	step.logger.Printf("Cache key: %s", key)
	step.logger.Printf("Cache paths:")
	step.logger.Printf(config.CachePaths)
	step.logger.Println()

	saver := cache.NewSaver(step.envRepo, step.logger, step.pathProvider, step.pathModifier, step.pathChecker)

	return saver.Save(cache.SaveCacheInput{
		StepId:           stepID,
		Verbose:          config.IsVerbose,
		Key:              key,
		Paths:            []string{config.CachePaths},
		IsKeyUnique:      true,
		CompressionLevel: config.CompressionLevel,
	})
}
