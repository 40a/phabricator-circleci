package main

import (
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type githubPusher struct {
	lock   sync.Mutex
	tmpDir string
	phab   *phabricatorConduit
	cc     *circleClient
}

func (p *githubPusher) setupRepository(ctx context.Context, url string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	repoDir, err := cloneDir(url)
	if err != nil {
		return wraperr(err, "cannot find directory to clone into")
	}
	l := getLog(ctx)
	ultimateDir := filepath.Join(p.tmpDir, repoDir)
	if _, err := os.Stat(ultimateDir); err == nil {
		// Already exists
		return nil
	}
	cloneCmd := exec.Command("git", "clone", url)
	cloneCmd.Dir = p.tmpDir
	l.Printf("Running command %#v", cloneCmd)
	cmdBytes, err := cloneCmd.CombinedOutput()
	if err != nil {
		return wraperr(err, "cannot clone repository %s: %s", url, string(cmdBytes))
	}
	_, err = os.Stat(ultimateDir)
	return err
}

func cloneDir(uri string) (string, error) {
	parts1 := strings.Split(uri, "/")
	if len(parts1) != 2 {
		return "", fmt.Errorf("Unable to make uri out of %s", uri)
	}
	parts2 := strings.Split(parts1[1], ".")
	if len(parts2) != 2 {
		return "", fmt.Errorf("Unable to make uri out of %s", uri)
	}
	return parts2[0], nil
}

func circleProject(uri string) (string, error) {
	parts1 := strings.Split(uri, ":")
	if len(parts1) != 2 {
		return "", fmt.Errorf("Unable to make circle project out of %s", uri)
	}
	parts2 := strings.Split(parts1[1], ".")
	if len(parts2) != 2 {
		return "", fmt.Errorf("Unable to make circle project out of %s", uri)
	}
	return parts2[0], nil
}

func (p *githubPusher) updateRepository(ctx context.Context, repoName string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	l := getLog(ctx)
	ultimateDir := filepath.Join(p.tmpDir, repoName)
	if _, err := os.Stat(ultimateDir); err != nil {
		return wraperr(err, "cannot stat directory %s", ultimateDir)
	}
	cloneCmd := exec.Command("git", "fetch", "--all", "-a", "-v", "--tags")
	cloneCmd.Dir = ultimateDir
	l.Printf("Running command %#v", cloneCmd)
	cmdBytes, err := cloneCmd.CombinedOutput()
	if err != nil {
		return wraperr(err, "cannot update repository %s: %s", repoName, string(cmdBytes))
	}
	l.Printf("Result of fetch: %s", string(cmdBytes))
	return nil
}

func (p *githubPusher) pushOrigin(ctx context.Context, repoName string, pushString string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	l := getLog(ctx)
	ultimateDir := filepath.Join(p.tmpDir, repoName)
	if _, err := os.Stat(ultimateDir); err != nil {
		return wraperr(err, "cannot stat directory %s", ultimateDir)
	}
	pushCmd := exec.Command("git", "push", "--force", "origin", pushString)
	pushCmd.Dir = ultimateDir
	l.Printf("Running command %#v", pushCmd)
	cmdBytes, err := pushCmd.CombinedOutput()
	if err != nil {
		return wraperr(err, "cannot update repository %s: %s", repoName, string(cmdBytes))
	}
	return nil
}

func (p *githubPusher) removeTag(ctx context.Context, repoName string, tag string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	l := getLog(ctx)
	ultimateDir := filepath.Join(p.tmpDir, repoName)
	if _, err := os.Stat(ultimateDir); err != nil {
		return wraperr(err, "cannot stat directory %s", ultimateDir)
	}
	pushCmd := exec.Command("git", "push", "origin", ":"+tag)
	pushCmd.Dir = ultimateDir
	l.Printf("Running command %#v", pushCmd)
	cmdBytes, err := pushCmd.CombinedOutput()
	if err != nil {
		return wraperr(err, "cannot update repository %s: %s", repoName, string(cmdBytes))
	}
	return nil
}

var errNotValidMessageType = errors.New("invalid message type")
