package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/errorutil"
	"github.com/bitrise-io/go-utils/v2/exitcode"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	xcodecache "github.com/bitrise-io/go-xcode/v2/xcodecache"
	"github.com/bitrise-steplib/bitrise-step-save-spm-cache/step"
)

func main() {
	exitCode := run()
	os.Exit(int(exitCode))
}

func run() exitcode.ExitCode {
	logger := log.NewLogger()
	envRepo := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepo)
	pathChecker := pathutil.NewPathChecker()
	pathProvider := pathutil.NewPathProvider()
	pathModifier := pathutil.NewPathModifier()
	derivedDataPathProvider := xcodecache.NewSwiftPackageCache()
	cacheStep := step.New(logger, inputParser, pathChecker, pathProvider, pathModifier, envRepo, derivedDataPathProvider)

	config, err := cacheStep.ProcessConfig()
	if err != nil {
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to process Step inputs: %w", err)))
		return exitcode.Failure
	}

	if runErr := cacheStep.Run(config); err != nil {
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to execute Step: %w", runErr)))
		return exitcode.Failure
	}

	return exitcode.Success
}
