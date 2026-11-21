package desk

import "testing"

// desk-model-availability AC-2: only a model whose answer names "tools" counts as able to call tools; an answer
// without a capabilities field says nothing and stays unknown.
func TestParseCapabilities(t *testing.T) {
	yes, no := true, false
	for _, c := range []struct {
		body string
		want *bool
	}{
		{`{"capabilities":["completion","tools"]}`, &yes},
		{`{"capabilities":["completion"]}`, &no},
		{`{"capabilities":[]}`, &no},
		{`{}`, nil},                  // older daemon: no capabilities field
		{`{"capabilities":null}`, nil}, // explicit unknown
	} {
		got, err := parseCapabilities([]byte(c.body))
		if err != nil {
			t.Errorf("%s: %v", c.body, err)
			continue
		}
		switch {
		case c.want == nil && got != nil:
			t.Errorf("%s: want unknown, got %v", c.body, *got)
		case c.want != nil && got == nil:
			t.Errorf("%s: want %v, got unknown", c.body, *c.want)
		case c.want != nil && got != nil && *got != *c.want:
			t.Errorf("%s: want %v, got %v", c.body, *c.want, *got)
		}
	}
	if _, err := parseCapabilities([]byte("kein JSON")); err == nil {
		t.Error("malformed answer must be an error")
	}
}

// desk-model-availability AC-2: the model names of /api/tags are read as they are.
func TestParseTags(t *testing.T) {
	names, err := parseTags([]byte(`{"models":[{"name":"qwen3:14b"},{"name":"nomic-embed-text"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "qwen3:14b" || names[1] != "nomic-embed-text" {
		t.Errorf("names %v", names)
	}
	if _, err := parseTags([]byte("{")); err == nil {
		t.Error("malformed answer must be an error")
	}
}

// desk-model-availability AC-6: the local model list keeps the configured models when Ollama reports nothing, and
// carries the capability only where it is known.
func TestMergeLocalModels(t *testing.T) {
	yes, no := true, false
	caps := map[string]*bool{"a": &yes, "c": &no}
	got := mergeLocalModels([]string{"a", "b"}, []string{"b", "c"}, caps)
	if len(got) != 3 {
		t.Fatalf("want three models without duplicates, got %+v", got)
	}
	if got[0].Name != "a" || got[1].Name != "b" || got[2].Name != "c" {
		t.Errorf("configured models first, then reported: %+v", got)
	}
	if got[0].Tools == nil || !*got[0].Tools {
		t.Errorf("a must be able to call tools: %+v", got[0])
	}
	if got[1].Tools != nil {
		t.Errorf("b is unknown, not %v", *got[1].Tools)
	}
	if got[2].Tools == nil || *got[2].Tools {
		t.Errorf("c must not claim tools: %+v", got[2])
	}

	// Ollama does not answer: the configured models stay visible.
	got = mergeLocalModels([]string{"qwen3:14b"}, nil, map[string]*bool{})
	if len(got) != 1 || got[0].Name != "qwen3:14b" || got[0].Tools != nil {
		t.Errorf("configured model must survive: %+v", got)
	}
}
