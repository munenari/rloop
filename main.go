package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
)

type (
	Runner struct {
		config Config
		max    int
	}
	Config struct {
		StatusFilename string   `toml:"status_file"`
		Command        []string `toml:"command"`
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
			log.Fatalln(fmt.Errorf("サンプル設定ファイルの書き込みに失敗しました: err: %w", err))
		}
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c := Config{}
	if _, err := toml.DecodeFile(*configFile, &c); err != nil {
		log.Fatalln(fmt.Errorf("設定ファイルの読み込みに失敗しました: err: %w", err))
	}
	x := Runner{config: c, max: *maxIteration}
	if err := x.Run(ctx); err != nil {
		log.Fatalln(fmt.Errorf("コマンドの実行に失敗しました: err: %w", err))
	}
}

func (x Runner) Run(ctx context.Context) error {
	if err := util.WriteStatusFile(x.config.StatusFilename, StatusRunning); err != nil {
		return err
	}
	commands, err := util.BuildPrompt(x.config.Command, x.config.Goal)
	if err != nil {
		return err
	}
	for range x.max {
	CHECKSTATUS:
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if err == context.Canceled {
				return nil
			}
			return err
		default:
			status := util.ReadStatusFile(x.config.StatusFilename)
			switch status {
			case StatusDone:
				return nil
			case StatusPause:
				time.Sleep(1 * time.Second)
				goto CHECKSTATUS
			default:
				if err := util.ExecuteCommand(ctx, commands); err != nil {
					return err
				}
			}
		}
	}
	return fmt.Errorf("繰り返し最大回数に到達しました (%d)", x.max)
}
