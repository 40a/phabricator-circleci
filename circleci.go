package main

import (
	"encoding/json"

	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sqs"
	"golang.org/x/net/context"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type circleCiMsg struct {
	AllParams  map[string]map[string]string `json:"allParamsJson"`
	FormParams circleMsg                    `json:"formparams"`

	originalMsg *sqs.Message
	parent      *circleManager
}

type circleMsg struct {
	Payload circleCiPayload `json:"payload"`
}

type circleManager struct {
	git  *githubPusher
	phab *phabricatorConduit
	ci   *circleClient
}

type circleCiPayload struct {
	BuildURL        string            `json:"build_url"`
	Branch          string            `json:"branch"`
	Outcome         string            `json:"outcome"`
	BuildTimeMS     int               `json:"build_time_millis"`
	VCSURL          string            `json:"vcs_url"`
	Reponame        string            `json:"reponame"`
	Username        string            `json:"username"`
	BuildNum        int               `json:"build_num"`
	BuildParameters map[string]string `json:"build_parameters"`
}

func (c *circleManager) parseCircleCImsg(msg *sqs.Message) (parsedMessage, error) {
	g := circleCiMsg{
		originalMsg: msg,
		parent:      c,
	}
	err := json.Unmarshal([]byte(*msg.Body), &g)
	if err != nil {
		return nil, err
	}
	if g.LooksValid() {
		return &g, nil
	}
	return nil, errNotValidMessageType
}

func (g *circleCiMsg) OriginalMsg() *sqs.Message {
	return g.originalMsg
}

func (g *circleCiMsg) diffIds() (int64, int64) {
	if g.FormParams.Payload.BuildParameters == nil {
		return 0, 0
	}
	diffStr, exists := g.FormParams.Payload.BuildParameters["diff"]
	if !exists {
		return 0, 0
	}

	revisionStr, exists := g.FormParams.Payload.BuildParameters["revision"]
	if !exists {
		return 0, 0
	}
	diff, err := strconv.ParseInt(diffStr, 10, 64)
	if err != nil {
		return 0, 0
	}
	revision, err := strconv.ParseInt(revisionStr, 10, 64)
	if err != nil {
		return 0, 0
	}
	return diff, revision
}

var diffResultTemplate = template.Must(template.New("").Parse(
	`| Build Result | Build time | Test count | Failing tests | Passing tests | Skipped Tests | Build Number
| ------------- | ---------- | ---------- | ------------- | ------------- | ------------  | ------------
| {{ .BuildResult }} | {{ .BuildTime }} | {{ .TestCount }}  | {{ .FailingTests }} | {{ .PassingTests }} | {{ .SkippedTests }} | {{ .BuildNumber }}

{{ if .Tests | len }}
(IMPORTANT) Some failing tests

{{range .Tests }}
| Classname | Test Name | Duration
| --------- | --------- | --------
| {{ .Classname }} | {{ .TestName }} | {{ .Duration }}

` + "```" + `
{{ .Message }}
` + "```" + `
{{ end }}{{ end }}`))

type diffResultStruct struct {
	BuildResult  string
	BuildTime    time.Duration
	TestCount    int
	FailingTests int
	PassingTests int
	SkippedTests int
	BuildNumber  int
	Tests        []diffResultTestStruct
}

type diffResultTestStruct struct {
	Classname string
	TestName  string
	Duration  time.Duration
	Message   string
}

func testResultMsg(cr circleTestResult) diffResultTestStruct {
	msg := ""
	if cr.Message != nil {
		msg = *cr.Message
	}
	if len(msg) > 300 {
		msg = msg[:299] + "... (trimmed output)"
	}
	return diffResultTestStruct{
		Classname: cr.Classname,
		TestName:  cr.Name,
		Duration:  time.Duration(int64(cr.RunTime * float64(time.Second.Nanoseconds()))),
		Message:   msg,
	}
}

func (g *circleCiMsg) populateTestResults(ctx context.Context) (diffResultStruct, []harbormasterUnitResult, error) {
	ciTestResults, err := g.parent.ci.testResults(ctx, g.FormParams.Payload.Username, g.FormParams.Payload.Reponame, g.FormParams.Payload.BuildNum)
	if err != nil {
		return diffResultStruct{}, nil, wraperr(err, "cannot get build results for %d", g.FormParams.Payload.BuildNum)
	}

	var unitTestResults []harbormasterUnitResult

	s := diffResultStruct{
		BuildResult: g.FormParams.Payload.Outcome,
		BuildTime:   time.Duration(int64(g.FormParams.Payload.BuildTimeMS) * time.Millisecond.Nanoseconds()),
		BuildNumber: g.FormParams.Payload.BuildNum,
		TestCount:   len(ciTestResults),
	}
	if len(ciTestResults) > 0 {
		unitTestResults = make([]harbormasterUnitResult, 0, len(ciTestResults))
		for _, circleTestResult := range ciTestResults {
			tr := harbormasterUnitResult{
				Name:      circleTestResult.Name,
				Namespace: circleTestResult.Classname,
				Duration:  &circleTestResult.RunTime,
				Engine:    circleTestResult.Source,
			}
			tr.Result = unitUnsound
			if circleTestResult.Result == "success" {
				tr.Result = unitPass
				s.PassingTests++
			}
			if circleTestResult.Result == "skipped" {
				tr.Result = unitSkip
				s.SkippedTests++
			}
			if circleTestResult.Result == "failure" || circleTestResult.Result == "error" {
				tr.Result = unitFail
				s.FailingTests++
				if len(s.Tests) < 3 {
					s.Tests = append(s.Tests, testResultMsg(circleTestResult))
				}
			}
			if circleTestResult.File != nil {
				tr.Path = *circleTestResult.File
			}
			unitTestResults = append(unitTestResults, tr)
		}
	}
	return s, unitTestResults, nil
}

func (g *circleCiMsg) harbormasterResult() harbormasterType {
	pt := harbormasterPass
	if g.FormParams.Payload.Outcome == "failed" {
		pt = harbormasterFail
	}
	if g.FormParams.Payload.Outcome == "canceled" {
		pt = harbormasterFail
	}
	return pt
}

func (g *circleCiMsg) Execute(ctx context.Context) error {
	l := getLog(ctx)
	l.Printf("Executing circleCI command for %s", g.FormParams.Payload.BuildURL)
	if !strings.HasPrefix(g.FormParams.Payload.Branch, "phabricator_test_") {
		l.Printf("circleci build isn't a phab attempt: %s", g.FormParams.Payload.Branch)
		return nil
	}

	diff, revision := g.diffIds()
	if diff == 0 || revision == 0 {
		l.Printf("Cannot parse diff/reivision out of %v", g.FormParams.Payload.BuildParameters)
		return nil
	}

	repoURI := g.FormParams.Payload.BuildParameters["staging_uri"]
	repoDir, err := cloneDir(repoURI)
	if err != nil {
		return wraperr(err, "cannot get clone directory")
	}

	msgStruct, unitTestResults, err := g.populateTestResults(ctx)
	if err != nil {
		return wraperr(err, "cannot create test results struct")
	}

	pt := g.harbormasterResult()

	buf := &bytes.Buffer{}
	if err := diffResultTemplate.Execute(buf, msgStruct); err != nil {
		return wraperr(err, "cannot build template for phab message")
	}

	if err := g.parent.phab.updateHarbormaster(ctx, g.FormParams.Payload.BuildParameters["phid"], pt, unitTestResults, nil); err != nil {
		return wraperr(err, "cannot post phab comment to %d", revision)
	}

	if err := g.parent.phab.createComment(ctx, int(revision), buf.String()); err != nil {
		return wraperr(err, "cannot post phab comment to %d", revision)
	}

	if err := g.parent.git.setupRepository(ctx, repoURI); err != nil {
		logIfErr(l, err, "cannot setup repository")
	}

	logIfErr(l, g.parent.git.removeTag(ctx, repoDir, fmt.Sprintf("phabricator_diff_branch_%d", diff)), "Cannot remove branch %d", diff)
	logIfErr(l, g.parent.git.removeTag(ctx, repoDir, g.FormParams.Payload.BuildParameters["staging_ref"]), "Cannot remove tag %d", diff)

	return nil
}

func (g *circleCiMsg) LooksValid() bool {
	return g.FormParams.Payload.Branch != "" && g.FormParams.Payload.BuildURL != "" && g.FormParams.Payload.Reponame != "" && g.FormParams.Payload.VCSURL != "" && g.FormParams.Payload.BuildParameters["phid"] != ""
}
