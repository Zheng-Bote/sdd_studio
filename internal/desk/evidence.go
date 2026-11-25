package desk

import "strings"

// Evidence statuses the desktop may show for one model and one guided flow.
const (
	EvidenceProven   = "proven"
	EvidenceFailed   = "failed"
	EvidenceUnproven = "unproven"
)

// JudgmentFlows are the guided flows whose answer is the model's own judgement (wörtliche Zitate, Widersprüche,
// Review). Only for them does the desktop state what has been tested — elsewhere it would be a claim without
// content.
var JudgmentFlows = []string{"sdd-conflicts", "sdd-review"}

// EvidenceEntry is one recorded test of a model with a guided flow.
type EvidenceEntry struct {
	Flow        string `json:"flow"`
	Protocol    string `json:"protocol"`
	ModelPrefix string `json:"model_prefix,omitempty"` // "" = every model of the provider
	Status      string `json:"status"`
	Note        string `json:"note"`
	Source      string `json:"source"`
}

// FlowEvidence is the recorded test state behind the README's "Nächste Schritte". It is a fixed table on purpose:
// a claim about a model is a claim someone tested, and this table is the only place the desktop takes one from.
// It grows only when a test is written down.
var FlowEvidence = []EvidenceEntry{
	{
		Flow:     "sdd-conflicts",
		Protocol: ProtocolClaude,
		Status:   EvidenceProven,
		Note:     "Szenario cert-expiry-watcher, vom leeren Ordner bis zu den Tests je Kriterium",
		Source:   "README, Nächste Schritte (Test mit Claude Sonnet 5.5)",
	},
	{
		Flow:        "sdd-conflicts",
		Protocol:    ProtocolOllama,
		ModelPrefix: "qwen3:",
		Status:      EvidenceFailed,
		Note:        "qwen3:14b erfand Pfade, Zeilen und Zitate; qwen3:32b fand nach rund 47 Minuten nichts",
		Source:      "README, Nächste Schritte (Gegentest mit lokalen Modellen)",
	},
}

// validEvidence says whether an entry may be shown: it must name a judgment flow, a known provider protocol, a
// status the desktop has a wording for and a source. An entry without a source is an unbacked claim, not evidence.
func validEvidence(e EvidenceEntry) bool {
	if strings.TrimSpace(e.Source) == "" || strings.TrimSpace(e.Protocol) == "" {
		return false
	}
	if e.Status != EvidenceProven && e.Status != EvidenceFailed {
		return false
	}
	for _, flow := range JudgmentFlows {
		if e.Flow == flow {
			return true
		}
	}
	return false
}

// EvidenceFor returns the recorded result for one flow and one model of a provider, taken from FlowEvidence.
//
// desk-model-availability AC-5
func EvidenceFor(protocol, model, flow string) (EvidenceEntry, bool) {
	return evidenceIn(FlowEvidence, protocol, model, flow)
}

// evidenceIn matches one model against a table. The most specific entry wins: a matching model prefix beats an entry
// for the whole protocol. Entries that are not valid evidence are ignored.
func evidenceIn(entries []EvidenceEntry, protocol, model, flow string) (EvidenceEntry, bool) {
	var best EvidenceEntry
	found := false
	for _, e := range entries {
		if !validEvidence(e) || e.Flow != flow || e.Protocol != protocol {
			continue
		}
		if e.ModelPrefix != "" && !strings.HasPrefix(model, e.ModelPrefix) {
			continue
		}
		if !found || len(e.ModelPrefix) > len(best.ModelPrefix) {
			best, found = e, true
		}
	}
	return best, found
}

// annotateEvidence fills model.Evidence for every judgment flow a model has a recorded result for. A model without
// a recorded result carries nothing; the desktop shows "unproven" for it and never "proven".
//
// desk-model-availability AC-4
func annotateEvidence(protocol string, models []modelView) {
	for i := range models {
		for _, flow := range JudgmentFlows {
			e, ok := EvidenceFor(protocol, models[i].Name, flow)
			if !ok {
				continue
			}
			if models[i].Evidence == nil {
				models[i].Evidence = map[string]EvidenceEntry{}
			}
			models[i].Evidence[flow] = e
		}
	}
}

// CLIEvidence is the recorded state of one agent CLI. Status covers the two ADR 0010 conditions (headless and MCP,
// effective write lock); ProxyProven covers the run against the desktop's own MCP endpoint. The two are separate on
// purpose: a documented headless contract says nothing about whether the CLI can talk to the proxy.
type CLIEvidence struct {
	CLI    string `json:"cli"`
	Status string `json:"status"`
	// ProxyProven is true only after a run against the desktop's endpoint is written down in Source.
	ProxyProven bool   `json:"proxy_proven"`
	Note        string `json:"note,omitempty"`
	Source      string `json:"source"`
}

// CLIEvidenceTable is the record behind the desktop's statements about an agent CLI. Like FlowEvidence it is a fixed
// table: a claim is a claim someone tested. An entry without a source counts as no entry.
//
// agenten-cli-proxy AC-8
var CLIEvidenceTable = []CLIEvidence{
	{
		CLI:         "claude",
		Status:      EvidenceProven,
		ProxyProven: true,
		Note:        "Headless, Schreibsperre, HTTP-Transport und Werkzeugaufruf durch ein Modell belegt (PROBE-OK:sdd_next, keine Verweigerung)",
		Source:      "docs/notes/headless-vertraege-agenten-clis.md (claude 2.1.330, Beleg 2026-11-25)",
	},
	{
		CLI:         "codewhale",
		Status:      EvidenceProven,
		ProxyProven: true,
		Note:        "Headless, OS-Schreibsperre und HTTP-Transport belegt; ein Werkzeugaufruf durch ein Modell ist nicht belegt — ohne Cloud-Schlüssel nutzt codewhale lokales Ollama, und das Modell erfand das Ergebnis, ohne das Werkzeug aufzurufen",
		Source:      "docs/notes/headless-vertraege-agenten-clis.md (codewhale 0.10.3, Beleg 2026-11-25)",
	},
	{
		CLI:    "agy",
		Status: EvidenceUnproven,
		Note:   "keine wirksame Schreibsperre belegt (Modus, der Schreiben sperrt, sperrt auch MCP)",
		Source: "docs/notes/headless-vertraege-agenten-clis.md (agy 1.5.9)",
	},
}

// validCLIEvidence says whether an entry may be shown: a known status and a source. Without a source it is an
// unbacked claim, not evidence.
func validCLIEvidence(e CLIEvidence) bool {
	if strings.TrimSpace(e.Source) == "" || strings.TrimSpace(e.CLI) == "" {
		return false
	}
	return e.Status == EvidenceProven || e.Status == EvidenceFailed
}

// EvidenceForCLI returns the recorded state of one CLI. A CLI without a valid entry is "unproven" with an empty
// note — never "proven".
//
// agenten-cli-proxy AC-8
func EvidenceForCLI(cli string) CLIEvidence {
	for _, e := range CLIEvidenceTable {
		if e.CLI == cli && validCLIEvidence(e) {
			return e
		}
	}
	return CLIEvidence{CLI: cli, Status: EvidenceUnproven}
}
