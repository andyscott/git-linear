package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func main() {

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

	fmt.Println("ISSUE", "\t", "BRANCH", "\t", "DESCRIPTION", "\000")
	for _, node := range resp.Data.Viewer.AssignedIssues.Nodes {
		fmt.Print(node.Identifier, "\t", node.BranchName, "\t", node.Description, "\000")
	}

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
