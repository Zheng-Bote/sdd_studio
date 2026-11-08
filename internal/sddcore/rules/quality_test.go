package rules

import "testing"

const qaPRD = "---\nfeature: billing\nstatus: active\n---\n# PRD\n\nQA-9 im Fließtext zählt nicht.\n\n## Qualitätsanforderungen\n\n" +
	"- **QA-1** Leistung: unter 5 s\n- **QA-2** Sicherheit: keine Klartext-Passwörter\n- ~~QA-3 entfallen~~\n- **QA-2** nochmal\n\n" +
	"## Akzeptanzkriterien\n\n- **AC-1** x, siehe QA-7\n"

// qa-trace AC-1, AC-2, AC-3: only the section counts, tombstones do not; Tech Spec and tests need the feature's slug;
// a repeated ID is marked.
func TestTraceQuality(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/prd/billing.md", qaPRD)
	writeFile(t, root, "docs/prd/other.md", "---\nfeature: other\nstatus: active\n---\n## Qualitätsanforderungen\n- **QA-1** fremd\n")
	writeFile(t, root, "docs/design/billing.md", "---\nfeature: billing\nstatus: active\n---\nSetzt QA-1 um.\nNochmal QA-1.\n")
	writeFile(t, root, "docs/design/other.md", "---\nfeature: other\nstatus: active\n---\nQA-2\n")
	writeFile(t, root, "a_test.go", "// billing QA-1: Laufzeit\n// QA-1 ohne Slug\n// other QA-1: fremd\n")
	cfg := featureConfig()
	cfg.Artifacts.PRD, cfg.Artifacts.Design = "docs/prd", "docs/design"
	got, err := TraceQuality(&Context{RepoRoot: root, Config: cfg}, "billing")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "QA-1" || got[1].ID != "QA-2" {
		t.Fatalf("ids: %+v", got)
	}
	q := got[0]
	if q.PRD != "docs/prd/billing.md" || q.Line != 11 || q.Text != "- **QA-1** Leistung: unter 5 s" || q.Duplicate {
		t.Errorf("QA-1: %+v", q)
	}
	if len(q.TechSpecs) != 1 || q.TechSpecs[0] != "docs/design/billing.md" || len(q.Tests) != 1 || q.Tests[0] != "a_test.go:1" {
		t.Errorf("references: %+v", q)
	}
	if !got[1].Duplicate || len(got[1].TechSpecs) != 0 || len(got[1].Tests) != 0 {
		t.Errorf("QA-2: %+v", got[1])
	}
	none, err := TraceQuality(&Context{RepoRoot: root, Config: cfg}, "fehlt")
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("no requirements must be an empty list: %v %v", none, err)
	}
}
