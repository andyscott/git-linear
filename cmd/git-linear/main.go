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
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
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

type LinearAPI struct {
	token string
}

func NewLinearAPI() (*LinearAPI, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get home dir: %w", err)
	}
	linearTokenData, err := ioutil.ReadFile(path.Join(homedir, ".linear_token"))
	if err != nil {
		return nil, fmt.Errorf("unable to read linear token: %w", err)
	}
	linearToken := strings.TrimSpace(string(linearTokenData))
	return &LinearAPI{
		token: linearToken,
	}, nil
}

func (api *LinearAPI) Request(jsonData map[string]interface{}) ([]byte, error) {
	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", "https://api.linear.app/graphql", bytes.NewBuffer(jsonValue))
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("bearer %s", api.token))
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("The HTTP request failed with error %w", err)
	}
	defer response.Body.Close()
	data, _ := ioutil.ReadAll(response.Body)
	return data, nil
}

func branch() error {

	linear, err := NewLinearAPI()
	if err != nil {
		return err
	}

	glam, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"fzf",
		"--ansi",
		"--header-lines=1",
		"--read0",
		"--delimiter=\t",
		"--with-nth=1,2",
		"--layout=reverse",
		"--preview-window=up:follow",
		"--preview=echo {3}",
		"--bind=enter:become(git checkout {2} 2>/dev/null || git checkout -b {2})",
	)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	data, err := linear.Request(map[string]interface{}{
		"query": tellMeAboutMyIssuesQuery,
	})
	if err != nil {
		return err
	}
	var resp tellMeAboutMyIssuesResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	io.WriteString(stdin, fmt.Sprint(
		"ISSUE", "\t",
		"BRANCH", "\t",
		"DESCRIPTION", "\000",
	))
	for _, node := range resp.Data.Viewer.AssignedIssues.Nodes {
		description, err := glam.Render(node.Description)
		if err != nil {
			return err
		}
		io.WriteString(stdin, fmt.Sprint(node.Identifier, "\t", node.BranchName, "\t", description, "\000"))
	}
	stdin.Close()

	cmd.Wait()

	return nil
}

const tellMeAboutMyIssuesQuery = `
query {
  viewer {
    assignedIssues(
      filter: {
        state: { type: { neq: "completed" } }
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

const urlFromBranchQuery = `
query($branchName: String!) {
  issueVcsBranchSearch(branchName: $branchName) {
    url
  }
}
`

type tellMeAboutMyIssuesResponse struct {
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

type urlFromBranchQueryResponse struct {
	Data struct {
		IssueVcsBranchSearch struct {
			URL string
		}
	} `json:"data"`
}

func open() error {
	linear, err := NewLinearAPI()
	if err != nil {
		return err
	}

	branchBytes, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return err
	}
	branch := strings.TrimSpace(string(branchBytes))

	fmt.Printf("Opening %s ...\n", branch)

	data, err := linear.Request(map[string]interface{}{
		"query": urlFromBranchQuery,
		"variables": map[string]string{
			"branchName": branch,
		},
	})
	if err != nil {
		return err
	}
	var resp urlFromBranchQueryResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	return openURL(resp.Data.IssueVcsBranchSearch.URL)
}

func openURL(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
