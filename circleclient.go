package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"io"
	"net/http"
)

type circleClient struct {
	token  string
	client http.Client
}

type scheduledBuild struct {
	Revision    string            `json:"revision"`
	BuildParams map[string]string `json:"build_parameters"`
}

type buildResponse struct {
	BuildURL string `json:"build_url"`
}

type circleTestResult struct {
	Classname  string  `json:"classname"`
	File       *string `json:"file"`
	Name       string  `json:"name"`
	Result     string  `json:"result"`
	RunTime    float64 `json:"run_time"`
	Message    *string `json:"message"`
	Source     string  `json:"source"`
	SourceType string  `json:"source_type"`
}

type circleTestGetResp struct {
	Tests []circleTestResult `json:"tests"`
}

func (c *circleClient) testResults(ctx context.Context, username string, project string, buildNum int) ([]circleTestResult, error) {
	url := fmt.Sprintf("https://circleci.com/api/v1/project/%s/%s/%d/tests?circle-token=%s", username, project, buildNum, c.token)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, wraperr(err, "cannot get req for url %s", url)
	}
	req.Header.Add("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, wraperr(err, "cannot GET request %s", url)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		getLog(ctx).Printf("Invalid status %d", resp.StatusCode)
		return nil, fmt.Errorf("non 200 response %d on %s", resp.StatusCode, url)
	}
	fullBody := bytes.Buffer{}
	_, err = io.Copy(&fullBody, resp.Body)
	if err != nil {
		return nil, wraperr(err, "cannot copy msg out of http body")
	}
	var r circleTestGetResp
	if err := json.NewDecoder(&fullBody).Decode(&r); err != nil {
		return nil, wraperr(err, "cannot decode JSON body")
	}
	return r.Tests, nil
}

func (c *circleClient) scheduleBuild(ctx context.Context, revision string, project string, tree string, buildParams map[string]string) (*buildResponse, error) {
	url := fmt.Sprintf("https://circleci.com/api/v1/project/%s/tree/%s?circle-token=%s", project, tree, c.token)
	b := &scheduledBuild{
		Revision:    revision,
		BuildParams: buildParams,
	}
	body := bytes.Buffer{}
	if err := json.NewEncoder(&body).Encode(b); err != nil {
		return nil, wraperr(err, "cannot encode build JSON")
	}
	getLog(ctx).Printf("Body: %s", body.String())
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, wraperr(err, "cannot make request to %s", url)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, wraperr(err, "cannot POST request to %s", url)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		getLog(ctx).Printf("Build params %v", buildParams)
		return nil, fmt.Errorf("non 201 response %d on %s", resp.StatusCode, url)
	}

	respBody := buildResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, wraperr(err, "response body does not look like JSON")
	}
	return &respBody, nil
}
