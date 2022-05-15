package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kevinsj/ado-pr/internal"
	"github.com/urfave/cli/v2"
)

func prCommands() []*cli.Command {
	//hacky way to gather some default values
	currDir, _ := os.Executable()
	repoName := filepath.Base(filepath.Dir(currDir))
	currBranch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	//branch and refs are different things, ado always need ref not just branch
	currBranchRef := "refs/heads/" + strings.TrimSpace(string(currBranch))

	return []*cli.Command{
		{
			Name:   "list",
			Usage:  "List all current pr in current repo",
			Action: internal.ListPrs(),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "name, n",
					Aliases:     []string{"n"},
					DefaultText: "repo name",
					Usage:       "Name of the repository, default to current dir",
					Value:       repoName,
				},
			},
		},
		{
			Name:   "create",
			Usage:  "Create pr",
			Action: internal.CreatePr(),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "name, n",
					Aliases:     []string{"n"},
					DefaultText: "repo name",
					Usage:       "Name of the repository, default to current dir",
					Value:       repoName,
				},
				&cli.StringFlag{
					Name:     "message, m",
					Aliases:  []string{"m"},
					Usage:    "Name of the repository, default to current dir",
					Required: true,
				},
				&cli.BoolFlag{
					Name:    "draft, d",
					Aliases: []string{"d"},
					Usage:   "Whether the pr is created as draft",
				},
				&cli.StringFlag{
					Name:     "source, s",
					Aliases:  []string{"s"},
					Usage:    "The head branch in [OWNER:]BRANCH format. Defaults to the currently checked out branch",
					Value:    currBranchRef,
					Required: false,
				},
				&cli.StringFlag{
					Name:     "target, t",
					Aliases:  []string{"t"},
					Usage:    "The base branch in the [OWNER:]BRANCH format. Defaults to the default branch of the upstream repository (usually master)",
					Required: false,
				},
			},
		},
	}
}

func main() {
	app := &cli.App{
		Name:        "ado-pr",
		Description: "A quick and easy way to list and create prs to Azure DevOps in CLI",
		Authors: []*cli.Author{
			{
				Name: "Kevin Jiang",
			},
		},
	}

	app.Before = internal.GetAzureClient()
	app.Commands = append(app.Commands, prCommands()...)

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			EnvVars: []string{"AZ_DEVOPS_TOKEN"},
			Name:    "token, t",
			Usage:   "Your azure devops token",
		},
		&cli.StringFlag{
			EnvVars: []string{"AZ_DEVOPS_URL"},
			Name:    "az-url, g",
			Usage:   "Base AZ_DEVOPS URL",
			Value:   "https://dev.azure.com/kev0709",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
