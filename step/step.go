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

	// xceleratedKey is used when Build Cache for Xcode relocated the SPM checkouts. The archive then
	// holds a different absolute path than the default-layout one, and restore replays archives to
	// their recorded paths, so the two layouts must not share a key namespace.
	xceleratedKey = `{{ .OS }}-{{ .Arch }}-spm-cache-xcelerate-{{ checksum "**/Package.resolved" }}`

	// EnvSwiftPackagesPath is exported by `bitrise-build-cache activate xcode`. When set, the
	// xcodebuild wrapper passes it to xcodebuild as -clonedSourcePackagesDirPath, which moves SPM
	// checkouts out of DerivedData and therefore out of this step's default path.
	EnvSwiftPackagesPath = "BITRISE_XCODE_SOURCE_PACKAGES_PATH"

	// defaultDerivedDataPath mirrors the derived_data_path default in step.yml. Bitrise materialises
	// input defaults into the environment, so this is the only way to tell "user left it alone" from
	// "user deliberately set this exact value" — the latter is rare and still honoured either way.
	defaultDerivedDataPath = "~/Library/Developer/Xcode/DerivedData/**"
)

type Input struct {
	Verbose          bool   `env:"verbose,required"`
	DerivedDataPath  string `env:"derived_data_path"`
	ProjectPath      string `env:"project_path"`
	CompressionLevel int    `env:"compression_level,range[1..19]"`
}

type Config struct {
	CachePaths       string
	Key              string
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

	if relocated := step.xcelerateSourcePackagesPath(input.DerivedDataPath); relocated != "" {
		return Config{
			CachePaths:       relocated,
			Key:              xceleratedKey,
			IsVerbose:        input.Verbose,
			CompressionLevel: input.CompressionLevel,
		}, nil
	}

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
		Key:              key,
		IsVerbose:        input.Verbose,
		CompressionLevel: input.CompressionLevel,
	}, nil
}

// xcelerateSourcePackagesPath returns the SPM checkout dir published by Build Cache for Xcode, or
// empty when it is not in play. Build Cache for Xcode moves DerivedData under ~/.bitrise/cache, and
// SPM checkouts follow it, so caching this step's default path would archive a directory the build
// never reads — a silent no-op that leaves every build re-resolving packages from origin.
// An explicitly overridden derived_data_path wins, so existing opted-in setups keep working.
func (step SaveCacheStep) xcelerateSourcePackagesPath(derivedDataPath string) string {
	relocated := strings.TrimSpace(step.envRepo.Get(EnvSwiftPackagesPath))
	if relocated == "" {
		return ""
	}

	if derivedDataPath != "" && derivedDataPath != defaultDerivedDataPath {
		step.logger.Warnf("Build Cache for Xcode relocated the SPM checkouts to %s, but derived_data_path is set to %s.", relocated, derivedDataPath)
		step.logger.Warnf("Caching %s as requested — clear derived_data_path to cache the relocated checkouts instead.", derivedDataPath)

		return ""
	}

	step.logger.Printf("Build Cache for Xcode is active, caching the relocated SPM checkouts (%s=%s)", EnvSwiftPackagesPath, relocated)

	return relocated
}

func (step SaveCacheStep) Run(config Config) error {
	step.logger.Println()
	step.logger.Printf("Cache key: %s", config.Key)
	step.logger.Printf("Cache paths:")
	step.logger.Printf(config.CachePaths)
	step.logger.Println()

	saver := cache.NewSaver(step.envRepo, step.logger, step.pathProvider, step.pathModifier, step.pathChecker, nil)

	return saver.Save(cache.SaveCacheInput{
		StepId:           stepID,
		Verbose:          config.IsVerbose,
		Key:              config.Key,
		Paths:            []string{config.CachePaths},
		IsKeyUnique:      true,
		CompressionLevel: config.CompressionLevel,
	})
}
