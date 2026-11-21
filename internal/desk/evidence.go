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
