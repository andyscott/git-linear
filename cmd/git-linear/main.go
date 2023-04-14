package main

import (
	"bufio"
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
	"syscall"
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

func previewLoop(glam *glamour.TermRenderer, data tellMeAboutMyIssuesResponse, rPipeFile string, wPipeFile string) error {
	r, err := os.OpenFile(rPipeFile, os.O_RDWR, 0640)
	if err != nil {
		return err
	}
	defer r.Close()
	reader := bufio.NewReader(r)

	for {
		identifier, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		identifier = strings.TrimSpace(identifier)
		w, err := os.OpenFile(wPipeFile, os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}

		for _, node := range data.Data.Viewer.AssignedIssues.Nodes {
			if node.Identifier == identifier {
				description, err := glam.Render(node.Description)
				if err != nil {
					return err
				}
				w.WriteString(description)
			}
		}
		w.Sync()
		w.Close()
	}
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
	tempDir, err := ioutil.TempDir("", "git-linear-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	rPipeFile := path.Join(tempDir, "r-pipe")
	wPipeFile := path.Join(tempDir, "w-pipe")
	err = syscall.Mkfifo(rPipeFile, 0666)
	if err != nil {
		return err
	}
	err = syscall.Mkfifo(wPipeFile, 0666)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"fzf",
		"--ansi",
		"--header-lines=1",
		"--read0",
		"--delimiter=\t",
		"--layout=reverse",
		"--preview-window=up:follow",
		fmt.Sprintf("--preview=echo {1} > %s; cat %s", rPipeFile, wPipeFile),
		"--bind=enter:become(git checkout {3} 2>/dev/null || git checkout -b {3})",
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
		"ID", "\t",
		"STATE", "\t",
		"BRANCH", "\000",
	))
	for _, node := range resp.Data.Viewer.AssignedIssues.Nodes {
		io.WriteString(stdin, fmt.Sprint(
			node.Identifier, "\t",
			node.State.Name, "\t",
			node.BranchName, "\000",
		))
	}
	stdin.Close()

	previewLoopDone := make(chan error)
	go func() {
		previewLoopDone <- previewLoop(glam, resp, rPipeFile, wPipeFile)
		close(previewLoopDone)
	}()

	cmdDone := make(chan error)
	go func() {
		cmdDone <- cmd.Wait()
		close(cmdDone)
	}()

	select {
	case err := <-previewLoopDone:
		return err
	case err := <-cmdDone:
		return err
	}
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
