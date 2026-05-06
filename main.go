package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	util "github.com/munenari/rloop/internal"
)

var (
	initCmd      = flag.Bool("init", false, "初期化コマンド（テンプレートを生成します）")
	configFile   = flag.String("c", ".rloop.toml", "設定ファイルのパス")
	maxIteration = flag.Int("max", 10, "繰り返し最大回数")

	//go:embed config.toml
	sampleConfig []byte

	statusPollingInterval = 5 * time.Second
)

type (
	Runner struct {
		config Config
		max    int
	}
	Config struct {
		StatusFilename string   `toml:"status_file"`
		Command        []string `toml:"command"`
		VerifyCommand  []string `toml:"verify_command"`
		NotifyCommand  []string `toml:"notify_command"`
		Goal           string   `toml:"goal"`
	}
)

const (
	StatusDone    = "DONE"
	StatusPause   = "PAUSE"
	StatusRunning = "RUNNING"
)

func main() {
	flag.Parse()
	if *initCmd {
		if err := os.WriteFile(*configFile, sampleConfig, os.ModePerm); err != nil {
			log.Fatalln(fmt.Errorf("rloop: サンプル設定ファイルの書き込みに失敗しました: err: %w", err))
		}
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c := Config{}
	if _, err := toml.DecodeFile(*configFile, &c); err != nil {
		log.Fatalln(fmt.Errorf("rloop: 設定ファイルの読み込みに失敗しました: err: %w", err))
	}
	if c.StatusFilename == "" {
		log.Fatalln(fmt.Errorf("rloop: ステータス管理ファイル名が空です"))
	}
	x := Runner{config: c, max: *maxIteration}
	if err := x.Run(ctx); err != nil {
		log.Fatalln(fmt.Errorf("rloop: コマンドの実行に失敗しました: err: %w", err))
	}
}

func (x Runner) Run(ctx context.Context) error {
	if err := util.WriteStatusFile(x.config.StatusFilename, StatusRunning); err != nil {
		return err
	}
	for i := range x.max {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.Canceled {
				return nil
			}
			return err
		default:
			status := util.ReadStatusFile(x.config.StatusFilename)
			fmt.Printf("rloop: current status: %s, count: %d\n", status, i)
			switch status {
			case StatusDone:
				done, err := x.processDone(ctx)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
			case StatusPause:
				if err := x.processPause(ctx); err != nil {
					return err
				}
			default:
				if err := x.processRun(ctx); err != nil {
					return err
				}
			}
		}
	}
	return fmt.Errorf("繰り返し最大回数に到達しました (%d)", x.max)
}

func (x Runner) processRun(ctx context.Context) error {
	prompt, err := util.BuildPrompt(x.config.Goal, x.config.StatusFilename, "")
	if err != nil {
		return err
	}
	fmt.Println(strings.Join([]string{"prompt:", "----", prompt, "----"}, "\n"))
	return util.ExecuteLLMCommand(ctx, x.config.Command, prompt)
}

func (x Runner) processPause(ctx context.Context) error {
	if res, err := util.ExecuteCommand(ctx, x.config.NotifyCommand); err != nil {
		return fmt.Errorf("failed to execute notify: %w\n%s", err, res)
	}
	t := time.NewTicker(statusPollingInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.Canceled {
				return nil
			}
			return err
		case <-t.C:
			if status := util.ReadStatusFile(x.config.StatusFilename); status != StatusPause {
				return nil
			}
		}
	}
}

func (x Runner) processDone(ctx context.Context) (done bool, err error) {
	res, err := util.ExecuteCommand(ctx, x.config.VerifyCommand)
	if err == nil {
		if res, err := util.ExecuteCommand(ctx, x.config.NotifyCommand); err != nil {
			return true, fmt.Errorf("failed to execute notify: %w\n%s", err, res)
		}
		return true, nil
	}
	if err := util.WriteStatusFile(x.config.StatusFilename, StatusRunning); err != nil {
		return true, fmt.Errorf("failed to execute verify: %w\n%s\nfailed to write status file: %w", err, res, err)
	}
	prompt, err := util.BuildPrompt(x.config.Goal, x.config.StatusFilename, string(res))
	if err != nil {
		return true, err
	}
	fmt.Println(strings.Join([]string{"prompt:", "----", prompt, "----"}, "\n"))
	err = util.ExecuteLLMCommand(ctx, x.config.Command, prompt)
	return false, err
}
