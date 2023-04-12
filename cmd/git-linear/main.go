package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

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
					branch()
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func branch() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to get home dir")
		os.Exit(2)
	}
	linearTokenData, err := ioutil.ReadFile(path.Join(homedir, ".linear_token"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to read linear token")
		os.Exit(2)
	}
	linearToken := strings.TrimSpace(string(linearTokenData))

	// If the user has glow installed, we can use that to help render previews.
	_, err = exec.LookPath("glow")
	var previewCommand string
	if err == nil {
		previewCommand = "echo {3} | glow"
	} else {
		previewCommand = "echo {3}"
	}

	cmd := exec.Command(
		"fzf",
		"--header-lines=1",
		"--read0",
		"--delimiter=\t",
		"--with-nth=1,2",
		"--layout=reverse",
		"--preview-window=up:follow",
		fmt.Sprintf("--preview=%s", previewCommand),
		"--bind=enter:become(git checkout {2} 2>/dev/null || git checkout -b {2})",
	)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open stdin pipe %s\n", err)
		os.Exit(2)
	}
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}

	jsonData := map[string]interface{}{
		"query":         tellMeAboutMyIssuesQuery,
		"operationName": "tellMeAboutMyIssues",
	}
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", "https://api.linear.app/graphql", bytes.NewBuffer(jsonValue))
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("bearer %s", linearToken))
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "The HTTP request failed with error %s\n", err)
		os.Exit(2)
	}
	defer response.Body.Close()
	data, _ := ioutil.ReadAll(response.Body)

	var resp Response
	err = json.Unmarshal(data, &resp)

	io.WriteString(stdin, fmt.Sprint("ISSUE", "\t", "BRANCH", "\t", "DESCRIPTION", "\000"))
	for _, node := range resp.Data.Viewer.AssignedIssues.Nodes {
		io.WriteString(stdin, fmt.Sprint(node.Identifier, "\t", node.BranchName, "\t", node.Description, "\000"))
	}
	stdin.Close()

	cmd.Wait()

}

const tellMeAboutMyIssuesQuery = `
query tellMeAboutMyIssues {
  viewer {
    assignedIssues(
      filter: {
        state: { name: { neq: "Done" } }
       }
    ) {
      nodes {
        identifier
        title
        description    
        state { name }
        branchName
      }
    }
  }
}`

type Response struct {
	Data struct {
		Viewer struct {
			AssignedIssues struct {
				Nodes []struct {
					Identifier  string
					Title       string
					Description string
					State       struct {
						Name string
					}
					BranchName string
				}
			}
		}
	} `json:"data"`
}
