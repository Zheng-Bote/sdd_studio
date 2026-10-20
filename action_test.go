package sddstudio_test

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type actionStep struct {
	Uses             string            `yaml:"uses"`
	Run              string            `yaml:"run"`
	WorkingDirectory string            `yaml:"working-directory"`
	Env              map[string]string `yaml:"env"`
}

// sdd-init-new AC-7: the action builds sdd from its own source, runs the gate, and passes inputs only via env.
func TestActionPassesInputsViaEnv(t *testing.T) {
	data, err := os.ReadFile("action.yml")
	if err != nil {
		t.Fatal(err)
	}
	var action struct {
		Inputs map[string]struct {
			Required bool `yaml:"required"`
		} `yaml:"inputs"`
		Runs struct {
			Using string       `yaml:"using"`
			Steps []actionStep `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(data, &action); err != nil {
		t.Fatal(err)
	}
	if action.Runs.Using != "composite" || !action.Inputs["gate"].Required {
		t.Fatalf("expected a composite action with required input gate, got %+v", action)
	}

	var built, gated bool
	for _, s := range action.Runs.Steps {
		if strings.Contains(s.Run, "${{") {
			t.Errorf("run script interpolates an expression; pass it via env instead:\n%s", s.Run)
		}
		if strings.Contains(s.Run, "go build") && strings.Contains(s.Run, "./cmd/sdd") && s.WorkingDirectory == "${{ github.action_path }}" {
			built = true
		}
		if strings.Contains(s.Run, `gate "$SDD_GATE"`) && s.Env["SDD_GATE"] == "${{ inputs.gate }}" {
			gated = true
			for _, forbidden := range []string{"|| true", "set +e", "continue-on-error"} {
				if strings.Contains(s.Run, forbidden) {
					t.Errorf("gate step must propagate the exit code, found %q", forbidden)
				}
			}
			if lines := strings.Split(strings.TrimSpace(s.Run), "\n"); !strings.Contains(lines[len(lines)-1], `"$RUNNER_TEMP/sdd"`) {
				t.Errorf("sdd must be the last command so its exit code is the step's exit code")
			}
		}
	}
	if !built || !gated {
		t.Fatalf("expected a build step in the action path and a gate step reading inputs from env (built=%v gated=%v)", built, gated)
	}
}
