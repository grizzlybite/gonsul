package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/grizzlybite/gonsul/internal/util"
)

func TestBuildConfigSuccess(t *testing.T) {
	flags := testConfigFlags()

	cfg, err := buildConfig(flags)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	if cfg.IsCloning() {
		t.Fatalf("IsCloning() = true, want false for local repo root without repo URL")
	}

	if got := cfg.GetLogLevel(); got != util.LogLevelDebug {
		t.Fatalf("GetLogLevel() = %d, want %d", got, util.LogLevelDebug)
	}

	if got := cfg.GetStrategy(); got != StrategyOnce {
		t.Fatalf("GetStrategy() = %q, want %q", got, StrategyOnce)
	}

	if got := cfg.GetValidExtensions(); !reflect.DeepEqual(got, []string{"json", "txt", "ini", "yaml"}) {
		t.Fatalf("GetValidExtensions() = %#v", got)
	}

	if !cfg.ShouldExpandJSON() {
		t.Fatalf("ShouldExpandJSON() = false, want true")
	}

	if !cfg.ShouldExpandYAML() {
		t.Fatalf("ShouldExpandYAML() = false, want true")
	}

	if got := cfg.AllowDeletes(); got != "skip" {
		t.Fatalf("AllowDeletes() = %q, want skip", got)
	}

	if !cfg.KeepFileExt() {
		t.Fatalf("KeepFileExt() = false, want true")
	}

	if got := cfg.GetHookAddr(); got != "127.0.0.1:9000" {
		t.Fatalf("GetHookAddr() = %q, want 127.0.0.1:9000", got)
	}

	if got := cfg.GetDryRunOutput(); got != DryRunOutputSummary {
		t.Fatalf("GetDryRunOutput() = %q, want %q", got, DryRunOutputSummary)
	}

	if cfg.WorkingChan() == nil {
		t.Fatalf("WorkingChan() = nil, want initialized channel")
	}
}

func TestBuildConfigVersionBypassesRequiredFlags(t *testing.T) {
	flags := testConfigFlags()
	*flags.Version = true
	*flags.ConsulURL = ""
	*flags.ValidExtensions = ""

	cfg, err := buildConfig(flags)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	if !cfg.IsShowVersion() {
		t.Fatalf("IsShowVersion() = false, want true")
	}
}

func TestBuildConfigRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(ConfigFlags)
		wantErr string
	}{
		{
			name: "missing consul url",
			mutate: func(flags ConfigFlags) {
				*flags.ConsulURL = ""
			},
			wantErr: "required flags not set",
		},
		{
			name: "missing input extensions",
			mutate: func(flags ConfigFlags) {
				*flags.ValidExtensions = ""
			},
			wantErr: "required flags not set",
		},
		{
			name: "invalid strategy",
			mutate: func(flags ConfigFlags) {
				*flags.Strategy = "later"
			},
			wantErr: "strategy invalid",
		},
		{
			name: "invalid allow deletes",
			mutate: func(flags ConfigFlags) {
				*flags.AllowDeletes = "sometimes"
			},
			wantErr: "AllowDelete method is invalid",
		},
		{
			name: "invalid log level",
			mutate: func(flags ConfigFlags) {
				*flags.LogLevel = "trace"
			},
			wantErr: "log level invalid",
		},
		{
			name: "invalid dry run output",
			mutate: func(flags ConfigFlags) {
				*flags.DryRunOutput = "verbose"
			},
			wantErr: "dry-run output invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := testConfigFlags()
			test.mutate(flags)

			cfg, err := buildConfig(flags)
			if err == nil {
				t.Fatalf("buildConfig() error = nil, want %q; cfg = %#v", test.wantErr, cfg)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("buildConfig() error = %q, want substring %q", err.Error(), test.wantErr)
			}
		})
	}
}

func TestBuildConfigLoadsSecretsMap(t *testing.T) {
	repoRoot := t.TempDir()
	secretFile := filepath.Join(repoRoot, "secrets.json")
	if err := os.WriteFile(secretFile, []byte(`{"db_password":"secret-value"}`), 0o600); err != nil {
		t.Fatalf("write secrets file: %v", err)
	}

	flags := testConfigFlags()
	*flags.RepoRootDir = repoRoot
	*flags.SecretsFile = "secrets.json"

	cfg, err := buildConfig(flags)
	if err != nil {
		t.Fatalf("buildConfig() error = %v", err)
	}

	if !cfg.DoSecrets() {
		t.Fatalf("DoSecrets() = false, want true")
	}

	if got := cfg.GetSecretsMap()["db_password"]; got != "secret-value" {
		t.Fatalf("GetSecretsMap()[db_password] = %q, want secret-value", got)
	}
}

func TestBuildConfigRejectsBadSecretsFile(t *testing.T) {
	tests := []struct {
		name       string
		fileName   string
		fileData   string
		wantErr    string
		createFile bool
	}{
		{
			name:       "missing file",
			fileName:   "missing.json",
			wantErr:    "cannot be found",
			createFile: false,
		},
		{
			name:       "invalid json",
			fileName:   "secrets.json",
			fileData:   `{not-json`,
			wantErr:    "could not parse keys JSON file",
			createFile: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			if test.createFile {
				if err := os.WriteFile(filepath.Join(repoRoot, test.fileName), []byte(test.fileData), 0o600); err != nil {
					t.Fatalf("write secrets file: %v", err)
				}
			}

			flags := testConfigFlags()
			*flags.RepoRootDir = repoRoot
			*flags.SecretsFile = test.fileName

			cfg, err := buildConfig(flags)
			if err == nil {
				t.Fatalf("buildConfig() error = nil, want %q; cfg = %#v", test.wantErr, cfg)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("buildConfig() error = %q, want substring %q", err.Error(), test.wantErr)
			}
		})
	}
}

func testConfigFlags() ConfigFlags {
	logLevel := util.LogDebug
	strategy := StrategyOnce
	repoURL := ""
	repoSSHKey := ""
	repoSSHUser := "git"
	repoBranch := "master"
	repoRemoteName := "origin"
	repoBasePath := "/"
	repoRootDir := "/tmp/gonsul-test-repo"
	consulURL := "http://127.0.0.1:8500"
	consulACL := "test-token"
	consulBasePath := "base"
	expandJSON := true
	expandYAML := true
	secretsFile := ""
	allowDeletes := "skip"
	pollInterval := 30
	validExtensions := "json,txt,ini,yaml"
	keepFileExt := true
	timeout := 10
	hookAddr := "127.0.0.1:9000"
	dryRunOutput := DryRunOutputSummary
	version := false

	return ConfigFlags{
		LogLevel:        &logLevel,
		Strategy:        &strategy,
		RepoURL:         &repoURL,
		RepoSSHKey:      &repoSSHKey,
		RepoSSHUser:     &repoSSHUser,
		RepoBranch:      &repoBranch,
		RepoRemoteName:  &repoRemoteName,
		RepoBasePath:    &repoBasePath,
		RepoRootDir:     &repoRootDir,
		ConsulURL:       &consulURL,
		ConsulACL:       &consulACL,
		ConsulBasePath:  &consulBasePath,
		ExpandJSON:      &expandJSON,
		ExpandYAML:      &expandYAML,
		SecretsFile:     &secretsFile,
		AllowDeletes:    &allowDeletes,
		PollInterval:    &pollInterval,
		ValidExtensions: &validExtensions,
		KeepFileExt:     &keepFileExt,
		Timeout:         &timeout,
		HookAddr:        &hookAddr,
		DryRunOutput:    &dryRunOutput,
		Version:         &version,
	}
}
