package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Authors: []*cli.Author{
			{
				Name:  "Andy Scott",
				Email: "andy.g.scott@gmail.com",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "branch",
				Aliases: []string{"b"},
				Usage:   "Switch to a branch for a linear ticket.",
				Action: func(cCtx *cli.Context) error {
					return branch()
				},
			},
			{
				Name:    "open",
				Aliases: []string{"o"},
				Usage:   "Opens the current linear ticket in your browser.",
				Action: func(cCtx *cli.Context) error {
					return open()
				},
			},
		},
		Description:          "Work with Linear and Git from the command line.",
		EnableBashCompletion: true,
	}
	cli.AppHelpTemplate = fmt.Sprintf(`%s
WEBSITE:
    https://github.com/andyscott/git-linear

`, cli.AppHelpTemplate)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
