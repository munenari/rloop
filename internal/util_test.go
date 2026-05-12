package util

import (
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	goal := "goal {{ .status_file }} {{ with .verify_result }}{{ . }}{{ end }}"
	res, err := BuildPrompt(goal, "dummy", "dummy2")
	if err != nil {
		t.Fatal(err)
	}
	if res != "goal dummy dummy2" {
		t.Error("unexpected result:", res)
	}
}
