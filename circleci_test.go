package main

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var exampleCirclePost = `{
    "allParamsJson": {
        "header": {
            "Accept-Encoding": "gzip, deflate",
            "CloudFront-Forwarded-Proto": "https",
            "CloudFront-Is-Desktop-Viewer": "true",
            "CloudFront-Is-Mobile-Viewer": "false",
            "CloudFront-Is-SmartTV-Viewer": "false",
            "CloudFront-Is-Tablet-Viewer": "false",
            "CloudFront-Viewer-Country": "US",
            "Via": "1.1 XYZ.cloudfront.net (CloudFront)",
            "X-Amz-Cf-Id": "XYZ==",
            "X-Forwarded-For": "54.215.24.241, 54.210.14.23",
            "X-Forwarded-Port": "443",
            "X-Forwarded-Proto": "https",
            "content-type": "application/json"
        },
        "path": {},
        "querystring": {}
    },
    "formparams": {
        "payload": {
            "all_commit_details": [
                {
                    "author_date": "2015-11-05T00:41:16-08:00",
                    "author_email": "a@signalfuse.com",
                    "author_login": "a",
                    "author_name": "A",
                    "body": "stuff",
                    "branch": "master",
                    "commit": "a3d45c9de1c26f97eefba1e06378a86b03d89846",
                    "commit_url": "https://github.com/signalfx/repo/commit/a3d45c9de1c26f97eefba1e06378a86b03d89846",
                    "committer_date": "2015-11-05T00:41:16-08:00",
                    "committer_email": "a@signalfuse.com",
                    "committer_login": "9park",
                    "committer_name": "A Username",
                    "subject": "Fix test flappiness by waiting for operation to complete."
                }
            ],
            "author_date": "2015-11-05T00:41:16-08:00",
            "author_email": "a@signalfuse.com",
            "author_name": "A Username",
            "body": "body",
            "branch": "master",
            "build_num": 254,
            "build_parameters": null,
            "build_time_millis": 456645,
            "build_url": "https://circleci.com/gh/signalfx/arepo/254",
            "canceled": false,
            "canceler": null,
            "circle_yml": {
                "string": "yaml"
            },
            "committer_date": "2015-11-05T00:41:16-08:00",
            "committer_email": "ausername@signalfuse.com",
            "committer_name": "A Username",
            "compare": "https://github.com/signalfx/arepo/compare/3e874878205e...a3d45c9de1c2",
            "dont_build": null,
            "failed": true,
            "has_artifacts": true,
            "infrastructure_fail": false,
            "is_first_green_build": false,
            "job_name": null,
            "lifecycle": "finished",
            "messages": [],
            "no_dependency_cache": null,
            "node": [
                {
                    "image_id": "circletar-1248-05fdb-20151015T182027Z",
                    "port": 64662,
                    "public_ip_addr": "54.204.207.201",
                    "ssh_enabled": null,
                    "username": "ubuntu"
                }
            ],
            "oss": false,
            "outcome": "failed",
            "owners": [],
            "parallel": 1,
            "previous": {
                "build_num": 253,
                "build_time_millis": 448817,
                "status": "failed"
            },
            "previous_successful_build": {
                "build_num": 232,
                "build_time_millis": 2399011,
                "status": "success"
            },
            "pull_request_urls": [],
            "queued_at": "2015-11-05T08:41:22.482Z",
            "reponame": "prototype",
            "retries": null,
            "retry_of": null,
            "ssh_enabled": false,
            "ssh_users": [],
            "start_time": "2015-11-05T08:41:22.490Z",
            "status": "failed",
            "steps": [
                {
                    "actions": [
                        {
                            "bash_command": null,
                            "canceled": null,
                            "continue": null,
                            "end_time": "2015-11-05T08:41:28.662Z",
                            "exit_code": null,
                            "failed": null,
                            "has_output": true,
                            "index": 0,
                            "infrastructure_fail": null,
                            "messages": [],
                            "name": "Starting the build",
                            "output_url": "https://outurl",
                            "parallel": false,
                            "run_time_millis": 5521,
                            "start_time": "2015-11-05T08:41:23.141Z",
                            "status": "success",
                            "step": 0,
                            "timedout": null,
                            "truncated": false,
                            "type": "infrastructure"
                        }
                    ],
                    "name": "Starting the build"
                }
            ],
            "stop_time": "2015-11-05T08:48:59.135Z",
            "subject": "Fix test flappiness by waiting for operation to complete.",
            "timedout": false,
            "usage_queued_at": "2015-11-05T08:41:21.987Z",
            "user": {
                "email": "i9parkk@gmail.com",
                "is_user": false,
                "login": "9park"
            },
            "username": "signalfuse",
            "vcs_revision": "a3d45c9de1c26f97eefba1e06378a86b03d89846",
            "vcs_tag": null,
            "vcs_url": "https://github.com/signalfx/arepo",
            "why": "github"
        }
    }
}

`

func TestParsingCircleCI(t *testing.T) {
	err := json.Unmarshal([]byte(exampleCirclePost), &circleCiMsg{})
	if err != nil {
		t.Errorf("Cannot parse json: %s", err.Error())
	}
}

func TestCompileTemplate(t *testing.T) {
	e := diffResultStruct{
		BuildResult:  "pass",
		BuildTime:    time.Minute * 2,
		TestCount:    1234,
		FailingTests: 1,
		PassingTests: 1233,
		BuildNumber:  123,
		Tests: []diffResultTestStruct{
			{
				Classname: "class",
				TestName:  "runStuff",
				Duration:  time.Second * 3,
				Message: `I fail
fail again
and again`,
			},
			{
				Classname: "class",
				TestName:  "runStuffAgain",
				Duration:  time.Second * 2,
				Message: `I fail
fail again
and again
and again
and again`,
			},
		},
	}
	buf := &bytes.Buffer{}
	assert.Nil(t, diffResultTemplate.Execute(buf, e))
	t.Log("*" + buf.String() + "*")
}
