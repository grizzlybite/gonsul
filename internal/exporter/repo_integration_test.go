package exporter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/grizzlybite/gonsul/internal/util"
)

type repoIntegrationConfig struct {
	expanderTestConfig
	repoURL     string
	repoRootDir string
}

func (c repoIntegrationConfig) IsCloning() bool           { return true }
func (c repoIntegrationConfig) GetRepoURL() string        { return c.repoURL }
func (c repoIntegrationConfig) GetRepoRootDir() string    { return c.repoRootDir }
func (c repoIntegrationConfig) GetRepoBasePath() string   { return "/" }
func (c repoIntegrationConfig) GetRepoBranch() string     { return "master" }
func (c repoIntegrationConfig) GetRepoRemoteName() string { return "origin" }
func (c repoIntegrationConfig) GetValidExtensions() []string {
	return []string{"txt"}
}

func TestExporterStartClonesLocalGitRepository(t *testing.T) {
	sourceDir := t.TempDir()
	cloneDir := filepath.Join(t.TempDir(), "clone")

	repo, err := git.PlainInit(sourceDir, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(sourceDir, "config.txt"), []byte("from git\n"), 0644); err != nil {
		t.Fatal(err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Add("config.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Commit("initial config", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Gonsul Test",
			Email: "gonsul@example.local",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatal(err)
	}

	exp := &exporter{
		config: repoIntegrationConfig{
			repoURL:     sourceDir,
			repoRootDir: cloneDir,
		},
		logger: util.NewLogger(0),
	}

	localData, err := exp.Start()
	if err != nil {
		t.Fatal(err)
	}

	if got := localData["config"]; got != "from git\n" {
		t.Fatalf("unexpected cloned config value: %q", got)
	}
}
