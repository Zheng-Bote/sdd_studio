package desk

import "testing"

// desk-model-availability AC-5: the recorded table is the only source of a claim; a model without an entry is not
// called proven, and the most specific entry wins.
func TestEvidenceFor(t *testing.T) {
	if e, ok := EvidenceFor(ProtocolClaude, "claude-sonnet-5.5", "sdd-conflicts"); !ok || e.Status != EvidenceProven {
		t.Errorf("claude with sdd-conflicts: %+v %v", e, ok)
	}
	if e, ok := EvidenceFor(ProtocolOllama, "qwen3:14b", "sdd-conflicts"); !ok || e.Status != EvidenceFailed {
		t.Errorf("qwen3:14b with sdd-conflicts: %+v %v", e, ok)
	}
	for _, c := range []struct{ protocol, model, flow string }{
		{ProtocolOllama, "llama3:8b", "sdd-conflicts"},  // untested model
		{ProtocolClaude, "claude-sonnet-5.5", "sdd-review"}, // untested flow
		{ProtocolOpenAI, "some/model", "sdd-conflicts"},     // untested provider
	} {
		if e, ok := EvidenceFor(c.protocol, c.model, c.flow); ok {
			t.Errorf("%v: expected no evidence, got %+v", c, e)
		}
	}
}

// desk-model-availability AC-5: a matching model prefix beats an entry for the whole protocol, and an entry without
// a source or with an unknown status is no evidence.
func TestEvidenceInMostSpecific(t *testing.T) {
	table := []EvidenceEntry{
		{Flow: "sdd-conflicts", Protocol: ProtocolOllama, Status: EvidenceProven, Note: "protocol wide", Source: "x"},
		{Flow: "sdd-conflicts", Protocol: ProtocolOllama, ModelPrefix: "qwen3:", Status: EvidenceFailed, Note: "prefix", Source: "x"},
		{Flow: "sdd-conflicts", Protocol: ProtocolOllama, ModelPrefix: "qwen3:32b", Status: EvidenceFailed, Note: "longer prefix", Source: "x"},
	}
	if e, ok := evidenceIn(table, ProtocolOllama, "qwen3:32b", "sdd-conflicts"); !ok || e.Note != "longer prefix" {
		t.Errorf("longest prefix must win: %+v %v", e, ok)
	}
	if e, ok := evidenceIn(table, ProtocolOllama, "qwen3:14b", "sdd-conflicts"); !ok || e.Note != "prefix" {
		t.Errorf("prefix must beat protocol: %+v %v", e, ok)
	}
	if e, ok := evidenceIn(table, ProtocolOllama, "llama3:8b", "sdd-conflicts"); !ok || e.Note != "protocol wide" {
		t.Errorf("protocol entry must apply: %+v %v", e, ok)
	}

	unbacked := []EvidenceEntry{
		{Flow: "sdd-conflicts", Protocol: ProtocolOllama, Status: EvidenceProven, Note: "no source"},
		{Flow: "sdd-conflicts", Protocol: ProtocolOllama, Status: "guess", Source: "x"},
		{Flow: "sdd-conflicts", Protocol: "", Status: EvidenceProven, Source: "x"},
		{Flow: "sdd-idea", Protocol: ProtocolOllama, Status: EvidenceProven, Source: "x"}, // not a judgment flow
	}
	for _, e := range unbacked {
		if got, ok := evidenceIn(unbacked, e.Protocol, "qwen3:14b", e.Flow); ok {
			t.Errorf("invalid entry %+v was used: %+v", e, got)
		}
	}
}

// desk-model-availability AC-4: a model gets evidence only for judgment flows that recorded a result; everything
// else stays absent so the desktop shows "unproven".
func TestAnnotateEvidence(t *testing.T) {
	models := []modelView{
		{Name: "claude-sonnet-5.5"},
		{Name: "qwen3:14b"},
		{Name: "llama3:8b"},
	}
	annotateEvidence(ProtocolClaude, models[:1])
	annotateEvidence(ProtocolOllama, models[1:])

	if got := models[0].Evidence["sdd-conflicts"].Status; got != EvidenceProven {
		t.Errorf("claude: %q", got)
	}
	if got := models[1].Evidence["sdd-conflicts"].Status; got != EvidenceFailed {
		t.Errorf("qwen3:14b: %q", got)
	}
	if len(models[2].Evidence) != 0 {
		t.Errorf("llama3:8b has no recorded result: %+v", models[2].Evidence)
	}
	if _, ok := models[0].Evidence["sdd-review"]; ok {
		t.Errorf("sdd-review has no recorded result")
	}
}

// desk-model-availability AC-5: every entry of the shipped table is backed by a source and a status the desktop has
// a wording for.
func TestFlowEvidenceIsValid(t *testing.T) {
	for _, e := range FlowEvidence {
		if !validEvidence(e) {
			t.Errorf("invalid entry: %+v", e)
		}
	}
}
