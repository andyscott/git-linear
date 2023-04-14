package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "branch",
				Aliases: []string{"b"},
				Usage:   "switch to a branch for a linear ticket",
				Action: func(cCtx *cli.Context) error {
					return branch()
				},
			},
			{
				Name:    "open",
				Aliases: []string{"o"},
				Usage:   "open a brower for the current branch's linear ticket",
				Action: func(cCtx *cli.Context) error {
					return open()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
