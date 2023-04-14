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
