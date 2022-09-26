package step

import (
	"fmt"

	"github.com/bitrise-io/go-steputils/v2/cache"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

const (
	stepId = "save-spm-cache"

	// Cache key template
	// OS + Arch: SPM works on Linux too, and Intel/ARM difference is important on macOS
	// checksum: Package.resolved is the dependency lockfile, either in the project root (pure Swift project)
	// or at project.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved
	key = `{{ .OS }}-{{ .Arch }}-spm-cache-debug-{{ checksum "**/Package.resolved" }}`

	// Cached path
	// The wildcard is for the unique project folder, such as `sample-swiftpm2-czkemcvuprosyehacrtonyiofjkk`
	path = "~/Library/Developer/Xcode/DerivedData/**/SourcePackages"
)

type Input struct {
	Verbose bool `env:"verbose,required"`
}

type SaveCacheStep struct {
	logger       log.Logger
	inputParser  stepconf.InputParser
	pathChecker  pathutil.PathChecker
	pathProvider pathutil.PathProvider
	pathModifier pathutil.PathModifier
	envRepo      env.Repository
}

func New(
	logger log.Logger,
	inputParser stepconf.InputParser,
	pathChecker pathutil.PathChecker,
	pathProvider pathutil.PathProvider,
	pathModifier pathutil.PathModifier,
	envRepo env.Repository,
) SaveCacheStep {
	return SaveCacheStep{
		logger:       logger,
		inputParser:  inputParser,
		pathChecker:  pathChecker,
		pathProvider: pathProvider,
		pathModifier: pathModifier,
		envRepo:      envRepo,
	}
}

func (step SaveCacheStep) Run() error {
	var input Input
	if err := step.inputParser.Parse(&input); err != nil {
		return fmt.Errorf("failed to parse inputs: %w", err)
	}
	stepconf.Print(input)
	step.logger.Println()

	step.logger.EnableDebugLog(input.Verbose)

	saver := cache.NewSaver(step.envRepo, step.logger, step.pathProvider, step.pathModifier, step.pathChecker)
	return saver.Save(cache.SaveCacheInput{
		StepId:  stepId,
		Verbose: input.Verbose,
		Key:     key,
		Paths:   []string{path},
	})
}
