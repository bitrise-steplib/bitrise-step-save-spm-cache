package step

import (
	"strings"
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

func TestXcelerateDerivedDataPath(t *testing.T) {
	const root = "/Users/vagrant/.bitrise/cache/xcode-dd"

	tests := []struct {
		name            string
		envValue        string
		derivedDataPath string
		want            string
	}{
		{
			name:            "relocated root used when derived_data_path is the default",
			envValue:        root,
			derivedDataPath: defaultDerivedDataPath,
			want:            root + "/*",
		},
		{
			name:            "relocated root used when derived_data_path is empty",
			envValue:        root,
			derivedDataPath: "",
			want:            root + "/*",
		},
		{
			name:            "explicit derived_data_path override wins",
			envValue:        root,
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
			step := newStepWithEnv(map[string]string{EnvDerivedDataPath: tt.envValue})

			if got := step.xcelerateDerivedDataPath(tt.derivedDataPath); got != tt.want {
				t.Errorf("xcelerateDerivedDataPath(%q) = %q, want %q", tt.derivedDataPath, got, tt.want)
			}
		})
	}
}

// Sharing a namespace would let one layout restore over the other. The xcelerate key must still
// sit under the default prefix, which is how the unchanged restore step finds it.
func TestKeyNamespaces(t *testing.T) {
	if !strings.HasPrefix(xceleratedKey, `{{ .OS }}-{{ .Arch }}-spm-cache-`) {
		t.Errorf("xcelerate key %q must stay under the restore step's prefix fallback", xceleratedKey)
	}
	if key == xceleratedKey {
		t.Fatalf("default and xcelerate cache keys must differ, both are %q", key)
	}
}
