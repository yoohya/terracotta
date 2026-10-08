package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/yoohya/terracotta/config"
)

func TestRunPlan(t *testing.T) {
	tests := []struct {
		name        string
		destroy     bool
		failOn      []string
		wantCalls   []terraformCall
		wantFailed  bool
		wantSummary string
	}{
		{
			name:    "plan runs in dependency order",
			destroy: false,
			wantCalls: []terraformCall{
				{Module: "a", Path: "a", Args: []string{"init", "-input=false"}},
				{Module: "a", Path: "a", Args: []string{"plan"}},
				{Module: "b", Path: "b", Args: []string{"init", "-input=false"}},
				{Module: "b", Path: "b", Args: []string{"plan"}},
				{Module: "c", Path: "c", Args: []string{"init", "-input=false"}},
				{Module: "c", Path: "c", Args: []string{"plan"}},
			},
			wantFailed:  false,
			wantSummary: "\nPlan Summary:\n✔ a: plan succeeded\n✔ b: plan succeeded\n✔ c: plan succeeded\n",
		},
		{
			name:    "plan --destroy runs plan -destroy in reverse order",
			destroy: true,
			wantCalls: []terraformCall{
				{Module: "c", Path: "c", Args: []string{"init", "-input=false"}},
				{Module: "c", Path: "c", Args: []string{"plan", "-destroy"}},
				{Module: "b", Path: "b", Args: []string{"init", "-input=false"}},
				{Module: "b", Path: "b", Args: []string{"plan", "-destroy"}},
				{Module: "a", Path: "a", Args: []string{"init", "-input=false"}},
				{Module: "a", Path: "a", Args: []string{"plan", "-destroy"}},
			},
			wantFailed:  false,
			wantSummary: "\nDestroy Plan Summary:\n✔ c: plan succeeded\n✔ b: plan succeeded\n✔ a: plan succeeded\n",
		},
		{
			name:    "plan --destroy continues after a failure",
			destroy: true,
			failOn:  []string{"b:plan"},
			wantCalls: []terraformCall{
				{Module: "c", Path: "c", Args: []string{"init", "-input=false"}},
				{Module: "c", Path: "c", Args: []string{"plan", "-destroy"}},
				{Module: "b", Path: "b", Args: []string{"init", "-input=false"}},
				{Module: "b", Path: "b", Args: []string{"plan", "-destroy"}},
				{Module: "a", Path: "a", Args: []string{"init", "-input=false"}},
				{Module: "a", Path: "a", Args: []string{"plan", "-destroy"}},
			},
			wantFailed:  true,
			wantSummary: "\nDestroy Plan Summary:\n✔ c: plan succeeded\n✖ b: failed - plan failed: exit status 1\n✔ a: plan succeeded\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := installFakeTerraform(t, tt.failOn...)
			var out bytes.Buffer

			failed := runPlan(&out, &config.Config{}, nodes("a", "b", "c"), tt.destroy, false)

			if failed != tt.wantFailed {
				t.Errorf("expected failed=%v, got %v", tt.wantFailed, failed)
			}
			if diff := cmp.Diff(tt.wantCalls, fake.calls); diff != "" {
				t.Errorf("terraform calls mismatch (-want +got):\n%s", diff)
			}
			if !strings.HasSuffix(out.String(), tt.wantSummary) {
				t.Errorf("expected summary %q, got:\n%s", tt.wantSummary, out.String())
			}
		})
	}
}

func TestRunPlan_DoesNotModifyInputOrder(t *testing.T) {
	installFakeTerraform(t)
	sorted := nodes("a", "b", "c")
	var out bytes.Buffer

	runPlan(&out, &config.Config{}, sorted, true, false)

	var got []string
	for _, n := range sorted {
		got = append(got, n.Path)
	}
	if diff := cmp.Diff([]string{"a", "b", "c"}, got); diff != "" {
		t.Errorf("input slice was modified (-want +got):\n%s", diff)
	}
}
