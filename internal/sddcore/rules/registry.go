package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sddstudio "github.com/Zheng-Bote/sdd_studio"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/catalog"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/config"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/report"
	"github.com/Zheng-Bote/sdd_studio/internal/sddcore/scan"
)

// Context holds all inputs required to evaluate a rule check.
type Context struct {
	RepoRoot    string
	Config      *config.Config
	Catalog     *catalog.Catalog
	CatalogPath string
	Gate        string // "" for default `sdd lint`, or "pre-implementation", "pre-merge", "pre-release"
	Feature     string
	Paths       []string
	Git         scan.Git         // nil means the git CLI in RepoRoot
	Now         func() time.Time // nil means time.Now

	// per-run caches, filled lazily by the feature checks
	diffDone      bool
	baseRef       string
	diff          []string
	diffErr       error
	artifactsDone bool
	artifacts     []scan.Artifact
	artifactsErr  error
}

// ReadHandbookFile reads a handbook file relative to RepoRoot if present on disk,
// falling back to the embedded handbook assets in the binary.
func (c *Context) ReadHandbookFile(relPath string) ([]byte, error) {
	cleanRel := filepath.ToSlash(filepath.Clean(relPath))
	diskPath := filepath.Join(c.RepoRoot, filepath.FromSlash(cleanRel))
	if data, err := os.ReadFile(diskPath); err == nil {
		return data, nil
	}
	return fs.ReadFile(sddstudio.EmbeddedAssets, cleanRel)
}

// CheckFunc executes a single rule check and returns raw findings. The runner fills severity (unless the check set a
// lower one explicitly, e.g. warn for "not assessable") and handbook_url.
type CheckFunc func(ctx *Context, rule catalog.Rule) ([]report.Finding, error)

type Registry struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

func NewRegistry() *Registry {
	r := &Registry{
		checks: make(map[string]CheckFunc),
	}
	registerBuiltins(r)
	return r
}

func (r *Registry) Register(name string, fn CheckFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[name] = fn
}

func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.checks[name]
	return ok
}

func (r *Registry) Get(name string) (CheckFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.checks[name]
	return fn, ok
}

// EffectiveRulesForGate computes Eff(g) according to section 8.3 and 9.5 (SDD-072):
// If gates.<gate>.override is non-empty in sdd.config.yaml, Eff(g) = override(g).
// Otherwise Eff(g) = gate_default(g) from the catalog (filtered by profile).
func EffectiveRulesForGate(cfg *config.Config, cat *catalog.Catalog, gate string) map[string]bool {
	eff := make(map[string]bool)
	override := cfg.GateOverrideFor(gate)
	if len(override) > 0 {
		for _, id := range override {
			eff[id] = true
		}
		return eff
	}
	for _, r := range cat.Rules {
		if !r.IsActiveInProfile(cfg.Profile) {
			continue
		}
		for _, g := range r.GateDefault {
			if g == gate {
				eff[r.ID] = true
				break
			}
		}
	}
	return eff
}

// Evaluate runs all active rules for ctx.Gate (or all profile-active rules if ctx.Gate == "").
func Evaluate(ctx *Context, reg *Registry) ([]report.Finding, error) {
	var gateSet map[string]bool
	if ctx.Gate != "" {
		gateSet = EffectiveRulesForGate(ctx.Config, ctx.Catalog, ctx.Gate)
	}

	var allFindings []report.Finding
	for _, rule := range ctx.Catalog.Rules {
		if !rule.IsActiveInProfile(ctx.Config.Profile) {
			continue
		}
		if ctx.Gate != "" && !gateSet[rule.ID] {
			continue
		}
		sev := rule.EffectiveSeverity(ctx.Gate)
		if sev == "" {
			// Section 9.3: If there is no entry for the current gate and no default,
			// the rule is not active in this gate.
			continue
		}

		fn, ok := reg.Get(rule.Check)
		if !ok {
			// SDD-071 will report unregistered checks; skip execution here.
			continue
		}

		findings, err := fn(ctx, rule)
		if err != nil {
			return nil, err
		}
		for i := range findings {
			findings[i].RuleID = rule.ID
			if findings[i].Severity == "" {
				findings[i].Severity = sev
			}
			if findings[i].HandbookURL == "" {
				findings[i].HandbookURL = rule.Handbook
			}
			if findings[i].ContentHash == "" {
				findings[i].ContentHash = HashContent(findings[i].Message)
			}
		}
		allFindings = append(allFindings, findings...)
	}

	return report.AssignOrdinalsAndSort(allFindings), nil
}

// HashContent computes the SHA-256 hex digest over normalized text (whitespace collapsed).
func HashContent(text string) string {
	norm := strings.Join(strings.Fields(text), " ")
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:])
}
