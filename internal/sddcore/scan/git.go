package scan

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Git is the read-only Git access the rule checks need. CLIGit is the production implementation; tests inject fakes.
type Git interface {
	// ChangedFiles returns the slash-separated paths changed in `git diff <base>...HEAD` (renames as delete + add).
	ChangedFiles(baseRef string) ([]string, error)
	// CurrentBranch returns the branch name, or "" if it cannot be determined (detached HEAD outside CI).
	CurrentBranch() (string, error)
	// ResolveBaseRef determines the merge base reference (concept section 8.3).
	ResolveBaseRef(configured string) (string, error)
}

// ErrGit wraps all failures of the Git CLI; the CLI maps it to exit code 2.
var ErrGit = errors.New("Git nicht verfügbar oder Aufruf fehlgeschlagen")

// CLIGit runs the git executable in Root.
type CLIGit struct {
	Root string
}

func (g CLIGit) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: git %s: %s", ErrGit, strings.Join(args, " "), strings.TrimSpace(stderr.String()+" "+err.Error()))
	}
	return stdout.String(), nil
}

func (g CLIGit) refExists(ref string) bool {
	_, err := g.run("rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

func (g CLIGit) ResolveBaseRef(configured string) (string, error) {
	if c := strings.TrimSpace(configured); c != "" {
		if !g.refExists(c) {
			return "", fmt.Errorf("%w: base_ref %q existiert nicht", ErrGit, c)
		}
		return c, nil
	}
	for _, env := range []string{"GITHUB_BASE_REF", "CI_MERGE_REQUEST_TARGET_BRANCH_NAME"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			for _, cand := range []string{"origin/" + v, v} {
				if g.refExists(cand) {
					return cand, nil
				}
			}
			return "", fmt.Errorf("%w: Basis-Branch %q aus $%s nicht gefunden (fetch-depth?)", ErrGit, v, env)
		}
	}
	if out, err := g.run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err == nil {
		if up := strings.TrimSpace(out); up != "" {
			return up, nil
		}
	}
	for _, cand := range []string{"main", "master"} {
		if g.refExists(cand) {
			return cand, nil
		}
	}
	return "", fmt.Errorf("%w: keine Merge-Basis ermittelbar (base_ref setzen)", ErrGit)
}

func (g CLIGit) ChangedFiles(baseRef string) ([]string, error) {
	out, err := g.run("diff", "--name-only", "--no-renames", "-z", baseRef+"...HEAD")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, f := range strings.Split(out, "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

func (g CLIGit) CurrentBranch() (string, error) {
	out, err := g.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if b := strings.TrimSpace(out); b != "HEAD" {
		return b, nil
	}
	for _, env := range []string{"GITHUB_HEAD_REF", "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", "CI_COMMIT_BRANCH"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v, nil
		}
	}
	return "", nil
}
