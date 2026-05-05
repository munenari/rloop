package util

import (
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	goal := "goal {{ .status_file }}"
	res, err := BuildPrompt(goal, "dummy")
	if err != nil {
		t.Fatal(err)
	}
	if res != "goal dummy" {
		t.Error("unexpected result:", res)
	}
}
