package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/charmbracelet/glamour"
)

func branch() error {

	linear, err := NewLinearAPI()
	if err != nil {
		return err
	}

	// As part of initialization glamour seems to send control characters to
	// the terminal. If we initialize glamour later, these characters may
	// wind up interfering with our terminal display.
	glam, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "git-linear-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// We want to live communicate with fzf as the user interacts but there
	// is no fzf API. Fortunately fzf is able to run shell commands on the,
	// so we can set up communication over named pipes.
	//
	// These pipes are named from our program's perspective:
	//   We will read from the read pipe and write to the write pipe.
	//   fzf will write to the read pipe and read from the write pipe.
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
		"--header=╱ ENTER checkout / CTRL-O open browser ╱\n\n",
		"--preview-window=up:follow",
		fmt.Sprintf("--preview=echo 'preview {1}' > %s; cat %s", rPipeFile, wPipeFile),
		"--bind=enter:become(git checkout {3} 2>/dev/null || git checkout -b {3})",
		fmt.Sprintf("--bind=ctrl-o:execute-silent(echo 'open {1}' > %s)", rPipeFile),
	)
	// fzf uses "$SHELL -c COMMAND" to launch the preview and become
	// functionality. We pin the shell to sh as a precautionary measure to
	// ensure our commands always work.
	_, err = exec.LookPath("sh")
	if err != nil {
		return err
	}
	cmd.Env = append(os.Environ(), "SHELL=sh")
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

	commandLoopDone := make(chan error)
	go func() {
		commandLoopDone <- commandLoop(glam, resp, rPipeFile, wPipeFile)
		close(commandLoopDone)
	}()

	cmdDone := make(chan error)
	go func() {
		cmdDone <- cmd.Wait()
		close(cmdDone)
	}()

	select {
	case err := <-commandLoopDone:
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
		url
		comments {
		  nodes {
		    user {
			  displayName
			}
			body
			createdAt
		  }
		}
      }
    }
  }
}`

type Issue struct {
	Identifier  string
	Title       string
	Description string
	State       struct {
		Name string
	}
	BranchName string
	URL        string
	Comments   struct {
		Nodes []struct {
			User struct {
				DisplayName string
			}
			Body      string
			CreatedAt string
		}
	}
}

type tellMeAboutMyIssuesResponse struct {
	Data struct {
		Viewer struct {
			AssignedIssues struct {
				Nodes []Issue
			}
		}
	} `json:"data"`
}

func commandLoop(
	glam *glamour.TermRenderer,
	data tellMeAboutMyIssuesResponse,
	rPipeFile string,
	wPipeFile string,
) error {
	r, err := os.OpenFile(rPipeFile, os.O_RDWR, 0640)
	if err != nil {
		return err
	}
	defer r.Close()
	reader := bufio.NewReader(r)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		parts := strings.Split(strings.TrimSpace(line), " ")

		command := parts[0]
		identifier := parts[1]
		var issue Issue
		for _, node := range data.Data.Viewer.AssignedIssues.Nodes {
			if node.Identifier == identifier {
				issue = node
				break
			}
		}
		if issue.Identifier == "" {
			return fmt.Errorf("unable to find issue for %s", identifier)
		}

		switch command {
		case "preview":

			w, err := os.OpenFile(wPipeFile, os.O_WRONLY|os.O_TRUNC, 0600)
			if err != nil {
				return err
			}

			lines := []string{
				"# " + issue.Title,
				"",
				issue.Description,
			}
			if len(issue.Comments.Nodes) > 0 {
				lines = append(lines,
					"# Activity",
					"",
				)
			}
			for _, n := range issue.Comments.Nodes {
				nameBit := "**" + n.User.DisplayName + ":**"
				timeBit := "_" + n.CreatedAt + "_"
				lines = append(lines,
					nameBit+strings.Repeat(" ", 80-len(nameBit)-len(timeBit))+timeBit,
					"",
					n.Body,
				)
			}
			blurb := strings.Join(lines, "\n")
			out, err := glam.Render(blurb)
			if err != nil {
				return err
			}
			w.WriteString(out)

			w.Sync()
			w.Close()

		case "open":
			openURL(issue.URL)
		}

	}
}
