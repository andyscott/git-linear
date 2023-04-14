package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

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

const urlFromBranchQuery = `
query($branchName: String!) {
  issueVcsBranchSearch(branchName: $branchName) {
    url
  }
}
`

type urlFromBranchQueryResponse struct {
	Data struct {
		IssueVcsBranchSearch struct {
			URL string
		}
	} `json:"data"`
}
