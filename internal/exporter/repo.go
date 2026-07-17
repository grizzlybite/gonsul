package exporter

import (
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"fmt"

	"github.com/grizzlybite/gonsul/internal/util"
)

// downloadRepo ...
func (e *exporter) downloadRepo() error {
	// Get some variables
	var (
		fileSystemPath = e.config.GetRepoRootDir()
		url            = e.config.GetRepoURL()
		sshUser        = e.config.GetRepoSSHUser()
		sshKey         = e.config.GetRepoSSHKey()
		auth           ssh.AuthMethod
	)

	// Check if SSH Key path was given
	if sshUser != "" && sshKey != "" {
		auth, _ = ssh.NewPublicKeysFromFile(sshUser, sshKey, "")
	}

	// Clone given repository
	repo, err := git.PlainClone(fileSystemPath, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Auth:              auth,
	})

	if err != nil {
		cloneError := util.RedactSensitive(err.Error(), e.config.GetRepoURL(), e.config.GetRepoSSHKey())
		e.logger.PrintDebug(fmt.Sprintf("REPO: failed clone (%s), trying to open directory", cloneError))

		// Cloning failed, most probably due to directory already cloned, moving to Open Dir
		repo, err = git.PlainOpen(e.config.GetRepoRootDir())

		if err != nil {
			return util.NewGonsulError(
				fmt.Errorf("REPO: failed clone and directory is not a git repo, try cleaning dir"),
				util.ErrorFailedCloning,
			)
		}

		e.logger.PrintDebug(fmt.Sprintf("REPO: git directory opened: %s", e.config.GetRepoRootDir()))
	}

	// We're still here, let's try to checkout required branch
	return e.tryCheckout(repo, &auth)
}

// tryCheckout ...
func (e *exporter) tryCheckout(repo *git.Repository, auth *ssh.AuthMethod) error {
	// Initiate our worktree
	workTree, err := repo.Worktree()
	if err := e.checkRepoError(err); err != nil {
		return err
	}

	// Get remotes, to check if current GIT is ours
	remotes, err := repo.Remotes()
	if err := e.checkRepoError(err); err != nil {
		return err
	}

	// Check if remote is valid (the same as ours
	if !e.checkIfRemoteValid(remotes) {
		return util.NewGonsulError(
			fmt.Errorf("REPO: remote url is not equal to provided repository URL"),
			util.ErrorFailedCloning,
		)
	}

	e.logger.PrintDebug(fmt.Sprintf("REPO: pulling changes: %s", e.config.GetRepoBranch()))
	// Pull can return non-fatal repository state errors; keep compatibility with
	// the previous behavior by logging them and continuing to checkout.
	err = workTree.Pull(&git.PullOptions{
		RemoteName: e.config.GetRepoRemoteName(),
		Auth:       *auth,
	})
	if err != nil {
		pullError := util.RedactSensitive(err.Error(), e.config.GetRepoURL(), e.config.GetRepoSSHKey())
		e.logger.PrintDebug(fmt.Sprintf("REPO: pull complete: %s", pullError))
	} else {
		e.logger.PrintDebug("REPO: pull complete")
	}

	e.logger.PrintDebug(fmt.Sprintf("REPO: checking out: %s", e.config.GetRepoBranch()))
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", e.config.GetRepoRemoteName(), e.config.GetRepoBranch())),
		Create: false,
		Force:  true,
	})
	return e.checkRepoError(err)
}

// checkIfRemoteValid ...
func (e *exporter) checkIfRemoteValid(remotes []*git.Remote) bool {
	// Iterate over remotes
	for _, remote := range remotes {
		// Iterate over URLs
		for _, url := range remote.Config().URLs {
			// Compare current url with ours
			if url == e.config.GetRepoURL() {
				return true
			}
		}
	}

	return false
}

// checkRepoError ...
func (e *exporter) checkRepoError(err error) error {
	if err != nil {
		message := util.RedactSensitive(err.Error(), e.config.GetRepoURL(), e.config.GetRepoSSHKey())
		return util.NewGonsulError(fmt.Errorf("REPO: %s", message), util.ErrorFailedCloning)
	}

	return nil
}
