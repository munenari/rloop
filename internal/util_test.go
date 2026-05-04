package util

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	in := []string{"opencode", "run", "{{ .prompt }}"}
	goal := "goal"
	res, err := BuildPrompt(in, goal)
	if err != nil {
		t.Fatal(err)
	}
	if v := strings.Join(res, " "); v != "opencode run goal" {
		t.Error("unexpected result:", v)
	}
}
