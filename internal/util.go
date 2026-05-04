package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

func ExecuteCommand(ctx context.Context, commands []string) error {
	if len(commands) == 0 {
		return fmt.Errorf("no commands")
	}
	name := commands[0]
	args := commands[1:]
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildPrompt(commands []string, goal string) ([]string, error) {
	res := make([]string, len(commands))
	v := map[string]any{
		"prompt": goal,
	}
	for i, s := range commands {
		buf := &bytes.Buffer{}
		tpl, err := template.New("").Parse(s)
		if err != nil {
			return res, err
		}
		if err := tpl.Execute(buf, v); err != nil {
			return res, err
		}
		res[i] = buf.String()
	}
	return res, nil
}

func ReadStatusFile(statusFilename string) string {
	data, err := os.ReadFile(statusFilename)
	if err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(string(data)))
}

func WriteStatusFile(statusFilename, status string) error {
	return os.WriteFile(statusFilename, []byte(status), os.ModePerm)
}
