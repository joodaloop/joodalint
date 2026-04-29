package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/joodaloop/hugolint/internal/config"
	"github.com/joodaloop/hugolint/internal/runner"

	_ "github.com/joodaloop/hugolint/internal/rules"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cfg, err := loadConfig("lint.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var n int
	switch os.Args[1] {
	case "md":
		if len(os.Args) >= 3 {
			cfg.Paths.MarkdownRoot = os.Args[2]
		}
		n, err = runner.Markdown(cfg)
	case "build":
		root := cfg.Paths.BuildRoot
		if len(os.Args) >= 3 {
			root = os.Args[2]
		}
		n, err = runner.Build(root)
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if n > 0 {
		os.Exit(1)
	}
}

func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return &config.Config{Paths: config.Paths{MarkdownRoot: "content", BuildRoot: "public"}}, nil
	}
	return nil, err
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lint md [dir] | lint build [dir]")
}
