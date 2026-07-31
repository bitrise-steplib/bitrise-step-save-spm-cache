package step

import (
	"testing"

	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

type fakeEnvRepo struct {
	env.Repository
	envs map[string]string
}

func (r fakeEnvRepo) Get(key string) string { return r.envs[key] }

func newStepWithEnv(envs map[string]string) SaveCacheStep {
	return SaveCacheStep{
		logger:  log.NewLogger(),
		envRepo: fakeEnvRepo{envs: envs},
	}
}

func TestXcelerateSourcePackagesPath(t *testing.T) {
	const relocated = "/Users/vagrant/.bitrise/cache/xcode-spm"

	tests := []struct {
		name            string
		envValue        string
		derivedDataPath string
		want            string
	}{
		{
			name:            "relocated path used when derived_data_path is the default",
			envValue:        relocated,
			derivedDataPath: defaultDerivedDataPath,
			want:            relocated,
		},
		{
			name:            "relocated path used when derived_data_path is empty",
			envValue:        relocated,
			derivedDataPath: "",
			want:            relocated,
		},
		{
			name:            "explicit derived_data_path override wins",
			envValue:        relocated,
			derivedDataPath: "/custom/dd",
			want:            "",
		},
		{
			name:            "no-op when Build Cache for Xcode is not active",
			envValue:        "",
			derivedDataPath: defaultDerivedDataPath,
			want:            "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := newStepWithEnv(map[string]string{EnvSwiftPackagesPath: tt.envValue})

			if got := step.xcelerateSourcePackagesPath(tt.derivedDataPath); got != tt.want {
				t.Errorf("xcelerateSourcePackagesPath(%q) = %q, want %q", tt.derivedDataPath, got, tt.want)
			}
		})
	}
}

// Sharing a namespace would let one layout restore over the other.
func TestKeyNamespacesDiffer(t *testing.T) {
	if key == xceleratedKey {
		t.Fatalf("default and xcelerate cache keys must differ, both are %q", key)
	}
}
