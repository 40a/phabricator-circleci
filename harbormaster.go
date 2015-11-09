package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sqs"
	"golang.org/x/net/context"
	"strconv"
)

type harbormasterPublisher struct {
	gp *githubPusher
}

type harbormasterMessage struct {
	AllParamTypes map[string]map[string]string `json:"allParamsJson"`

	gp          *githubPusher
	originalMsg *sqs.Message
}

func (p *harbormasterPublisher) parseHarbormasterMsg(msg *sqs.Message) (parsedMessage, error) {
	g := harbormasterMessage{
		originalMsg: msg,
		gp:          p.gp,
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

var _ msgConstructor = (&harbormasterPublisher{}).parseHarbormasterMsg

func (g *harbormasterMessage) OriginalMsg() *sqs.Message {
	return g.originalMsg
}

func (g *harbormasterMessage) Execute(ctx context.Context) error {
	l := getLog(ctx)
	repoURI := g.AllParamTypes["querystring"]["staging_uri"]
	repoDir, err := cloneDir(repoURI)
	if err != nil {
		return wraperr(err, "cannot find repo dir to execute harbormaster msg")
	}
	cp, err := circleProject(repoURI)
	if err != nil {
		return wraperr(err, "cannot find circle dir to execute harbormaster msg")
	}
	if err := g.gp.setupRepository(ctx, repoURI); err != nil {
		return wraperr(err, "cannot setup repository %s", g.AllParamTypes["querystring"]["staging_uri"])
	}
	if err := g.gp.updateRepository(ctx, repoDir); err != nil {
		return wraperr(err, "cannot update repository to latest version")
	}
	diffID := g.getDiffID()
	revID := g.getRevID()
	ref := g.AllParamTypes["querystring"]["staging_ref"]
	destBranch := fmt.Sprintf("phabricator_diff_branch_%d", diffID)
	pushString := fmt.Sprintf("%s:refs/heads/%s", ref, destBranch)
	if err := g.gp.pushOrigin(ctx, repoDir, pushString); err != nil {
		return wraperr(err, "cannot push tag to origin: %s", pushString)
	}

	resp, err := g.gp.cc.scheduleBuild(ctx, ref, cp,
		fmt.Sprintf("phabricator_test_%s", g.AllParamTypes["querystring"]["callsign"]),
		g.AllParamTypes["querystring"])
	if err != nil {
		return wraperr(err, "cannot post a scheduled bulid for %s", ref)
	}
	msg := fmt.Sprintf("Your revision is building in CirlceCI.  Build URL: %s", resp.BuildURL)

	err = g.gp.phab.createComment(
		ctx,
		revID,
		msg)
	logIfErr(l, err, "Unable to update diff, but moving on with life")
	return nil
}

func (g *harbormasterMessage) getDiffID() int {
	i, _ := strconv.ParseInt(g.AllParamTypes["querystring"]["diff"], 10, 64)
	return int(i)
}

func (g *harbormasterMessage) getRevID() int {
	i, _ := strconv.ParseInt(g.AllParamTypes["querystring"]["revision"], 10, 64)
	return int(i)
}

func (g *harbormasterMessage) LooksValid() bool {
	qs := g.AllParamTypes["querystring"]
	if qs == nil {
		return false
	}
	return len(qs["phid"]) > 0
}
