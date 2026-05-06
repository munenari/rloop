package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func ExecuteLLMCommand(ctx context.Context, commands []string, prompt string) error {
	if len(commands) == 0 {
		return fmt.Errorf("no commands")
	}
	name := commands[0]
	args := commands[1:]
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(prompt)
	return cmd.Run()
}

func ExecuteCommand(ctx context.Context, commands []string) ([]byte, error) {
	if len(commands) == 0 {
		return nil, nil
	}
	name := commands[0]
	args := commands[1:]
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func BuildPrompt(goal, statusFile, verifyResult string) (string, error) {
	v := map[string]any{
		"status_file":   statusFile,
		"verify_result": verifyResult,
	}
	buf := &bytes.Buffer{}
	tpl, err := template.New("").Parse(goal)
	if err != nil {
		return "", err
	}
	if err := tpl.Execute(buf, v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ReadStatusFile(statusFilename string) string {
	data, err := os.ReadFile(statusFilename)
	if err != nil {
		return ""
	}
	decoder := unicode.BOMOverride(unicode.UTF8.NewDecoder())
	reader := transform.NewReader(bytes.NewReader(data), decoder)
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(string(decoded)))
}

func WriteStatusFile(statusFilename, status string) error {
	return os.WriteFile(statusFilename, []byte(status), os.ModePerm)
}
