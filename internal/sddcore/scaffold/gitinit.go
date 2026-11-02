package scaffold

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// GitInitMode says what `sdd init --git` does with Git.
type GitInitMode int

const (
	// GitCreate: no repository yet; git init and a first commit of the created files follow the plan.
	GitCreate GitInitMode = iota
	// GitSkip: the folder already lies inside a repository; only the files are created.
	GitSkip
)

// GitSkipNote explains GitSkip to the user.
const GitSkipNote = "liegt bereits in einem Git-Repository; kein git init, kein Commit"

// InitBranch is the branch `sdd init --git` creates.
const InitBranch = "main"

// ErrGitInit marks an unmet precondition of `sdd init --git`; nothing has been written when it occurs.
var ErrGitInit = errors.New("sdd init --git nicht möglich")

// CheckGitInit decides before anything is written whether Git is initialized (ADR 0003, decision 8): only outside an
// existing repository, and only if git is installed and can name a committer.
func CheckGitInit(root string) (GitInitMode, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return GitCreate, fmt.Errorf("%w: git ist nicht installiert oder nicht im PATH", ErrGitInit)
	}
	g := scan.CLIGit{Root: root}
	if g.IsRepo() {
		return GitSkip, nil
	}
	if !g.HasIdentity() {
		return GitCreate, fmt.Errorf("%w: Git-Identität fehlt — git config --global user.name \"Ihr Name\" und git config --global user.email \"ihre@adresse\" setzen", ErrGitInit)
	}
	return GitCreate, nil
}

// CommitInit runs git init on branch main and commits exactly the files the plan created; it returns the short hash.
func CommitInit(root string, results []Result) (string, error) {
	var paths []string
	for _, r := range results {
		if r.Status == StatusCreated {
			paths = append(paths, r.Path)
		}
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("%w: keine Datei angelegt, nichts zu committen", ErrGitInit)
	}
	g := scan.CLIGit{Root: root}
	if err := g.InitRepo(InitBranch); err != nil {
		return "", err
	}
	return g.Commit("chore: sdd init", paths)
}
