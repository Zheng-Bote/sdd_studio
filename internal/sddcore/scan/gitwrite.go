package scan

import (
	"fmt"
	"strings"
)

// GitWriter is the writing Git access permitted by ADR 0001, used only by `sdd adopt --apply`:
// inspect the worktree, create and switch to a new branch, and commit explicit paths. No push, merge, rebase or reset.
type GitWriter interface {
	// IsRepo reports whether the root is inside a Git work tree.
	IsRepo() bool
	// DirtyPaths returns modified, staged and untracked paths (porcelain v1), empty for a clean worktree.
	DirtyPaths() ([]string, error)
	// CurrentBranch returns the checked-out branch, or "" for a detached HEAD.
	CurrentBranch() (string, error)
	// BranchExists reports whether refs/heads/<name> exists.
	BranchExists(name string) (bool, error)
	// CreateBranch validates the name, creates the branch at HEAD and switches to it.
	CreateBranch(name string) error
	// Commit stages exactly the given paths and commits only them; it returns the new commit's short hash.
	Commit(message string, paths []string) (string, error)
}

func (g CLIGit) IsRepo() bool {
	out, err := g.run("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

func (g CLIGit) DirtyPaths() ([]string, error) {
	out, err := g.run("status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var paths []string
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 4 {
			continue
		}
		paths = append(paths, e[3:])
		if e[0] == 'R' || e[0] == 'C' {
			i++ // the next entry is the source path of a rename or copy
		}
	}
	return paths, nil
}

func (g CLIGit) BranchExists(name string) (bool, error) {
	if _, err := g.run("check-ref-format", "--branch", name); err != nil {
		return false, fmt.Errorf("%w: ungültiger Branch-Name %q", ErrGit, name)
	}
	_, err := g.run("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil, nil
}

func (g CLIGit) CreateBranch(name string) error {
	if _, err := g.run("check-ref-format", "--branch", name); err != nil {
		return fmt.Errorf("%w: ungültiger Branch-Name %q", ErrGit, name)
	}
	_, err := g.run("switch", "--create", name)
	return err
}

func (g CLIGit) Commit(message string, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("%w: Commit ohne Pfade", ErrGit)
	}
	if _, err := g.run(append([]string{"add", "--"}, paths...)...); err != nil {
		return "", err
	}
	if _, err := g.run(append([]string{"commit", "--quiet", "-m", message, "--"}, paths...)...); err != nil {
		return "", err
	}
	out, err := g.run("rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
