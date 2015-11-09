package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type phabricatorConduit struct {
	apiToken string
	url      *url.URL
	client   http.Client
}

type queryResult struct {
	Result map[string]*diffObj `json:"result"`
}

func init() {}

type diffObj struct {
	AuthorName                string `json:"authorName"`
	ID                        string `json:"id"`
	RevisionID                string `json:"revisionID"`
	SourceControlBaseRevision string `json:"sourceControlBaseRevision"`
}

func (d *diffObj) String() string {
	return fmt.Sprintf("Rev [%s] AuthorName [%s] ID[%s] RevisionID[%s]", d.SourceControlBaseRevision, d.AuthorName, d.ID, d.RevisionID)
}

type createCommentResult struct {
	RevisionID string `json:"revision_id"`
	URI        string `json:"uri"`
}

type unitResult string

const (
	unitPass = unitResult("pass")
	unitFail = unitResult("fail")
	unitSkip = unitResult("skip")
	//	unitBroken  = unitResult("broken")
	unitUnsound = unitResult("unsound")
)

type harbormasterUnitResult struct {
	Name      string     `json:"name"`
	Result    unitResult `json:"result"`
	Namespace string     `json:"namespace,omitempty"`
	Engine    string     `json:"engine,omitempty"`
	Duration  *float64   `json:"duration,omitempty"`
	Path      string     `json:"path,omitempty"`
}

type lintResult struct {
}

type harbormasterType string

const (
	harbormasterPass = harbormasterType("pass")
	harbormasterFail = harbormasterType("fail")

//	HarbormasterWork = harbormasterType("work")
)

func (p *phabricatorConduit) updateHarbormaster(ctx context.Context, phid string, t harbormasterType, units []harbormasterUnitResult, lints []lintResult) error {
	u := *p.url
	u.Path = "/api/harbormaster.sendmessage"
	v := url.Values{}
	v.Add("api.token", p.apiToken)
	v.Add("buildTargetPHID", phid)
	v.Add("type", string(t))
	if len(units) > 0 {
		unitStr, err := json.Marshal(&units)
		if err != nil {
			return wraperr(err, "cannot marshall unit tests")
		}
		v.Add("unit", string(unitStr))
	}
	if len(lints) > 0 {
		lintStr, err := json.Marshal(&lints)
		if err != nil {
			return wraperr(err, "cannot marshall lints tests")
		}
		v.Add("lint", string(lintStr))
	}
	resp, err := p.client.PostForm(u.String(), v)
	if err != nil {
		return wraperr(err, "cannot POST comment")
	}
	getLog(ctx).Printf("Posted to URI %s", u.String())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code %d", resp.StatusCode)
	}

	return nil
}

func (p *phabricatorConduit) createComment(ctx context.Context, revisionID int, message string) error {
	u := *p.url
	u.Path = "/api/differential.createcomment"
	v := url.Values{}
	v.Add("api.token", p.apiToken)
	revisionIStr := strconv.FormatInt(int64(revisionID), 10)
	v.Add("revision_id", revisionIStr)
	v.Add("message", message)
	resp, err := p.client.PostForm(u.String(), v)
	if err != nil {
		return wraperr(err, "cannot POST comment")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code %d", resp.StatusCode)
	}
	queryRes := createCommentResult{}
	d := json.NewDecoder(resp.Body)
	if err := d.Decode(&queryRes); err != nil {
		return wraperr(err, "cannot decode response body")
	}
	getLog(ctx).Printf("Posted to URI %s", queryRes.URI)
	return nil
}

func (p *phabricatorConduit) revisionForDiff(ctx context.Context, diffid int) (int, error) {
	u := *p.url
	u.Path = "/api/differential.querydiffs"
	v := url.Values{}
	v.Add("api.token", p.apiToken)
	idStr := strconv.FormatInt(int64(diffid), 10)
	v.Add("ids[0]", idStr)
	resp, err := p.client.PostForm(u.String(), v)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("invalid status code %d", resp.StatusCode)
	}
	queryRes := queryResult{}
	bodyBuff := bytes.Buffer{}
	_, err = io.Copy(&bodyBuff, resp.Body)
	getLog(ctx).Printf("response len is %d", bodyBuff.Len())
	if err != nil {
		return 0, wraperr(err, "cannot read from response body")
	}
	d := json.NewDecoder(&bodyBuff)
	if err := d.Decode(&queryRes); err != nil {
		return 0, wraperr(err, "cannot decode response body")
	}
	obj, exists := queryRes.Result[idStr]
	if !exists || obj == nil {
		return 0, fmt.Errorf("cannot find diff for %d in queryRes %v obj %s", diffid, queryRes, obj.String())
	}
	if obj.RevisionID == "" {
		return 0, nil
	}
	parsedRev, err := strconv.ParseInt(obj.RevisionID, 10, 64)
	if err != nil {
		return 0, wraperr(err, "cannot parse revision %s", obj.RevisionID)
	}
	return int(parsedRev), nil
}
