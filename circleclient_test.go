package main

import (
	"encoding/json"
	"testing"
)

var response = `{
  "compare" : null,
  "previous_successful_build" : {
    "build_num" : 5,
    "status" : "success",
    "build_time_millis" : 886580
  },
  "build_parameters" : { },
  "oss" : false,
  "committer_date" : "2015-10-28T19:44:40Z",
  "body" : "commit body",
  "usage_queued_at" : "2015-10-29T05:58:39.873Z",
  "retry_of" : null,
  "reponame" : "areponame",
  "ssh_users" : [ ],
  "build_url" : "https://circleci.com/gh/signalfx/areponame/20",
  "parallel" : 1,
  "failed" : null,
  "branch" : "phabricator",
  "username" : "signalfx",
  "author_date" : "2015-10-28T19:44:07Z",
  "why" : "api",
  "user" : {
    "is_user" : true,
    "login" : "cep21",
    "name" : "Jack Lindamood",
    "email" : "jack@signalfx.com",
    "avatar_url" : "https://avatars.githubusercontent.com/u/20358?v=3"
  },
  "vcs_revision" : "3a6bfed57ea1d93f9f2b332c93786b30f68e7d3e",
  "vcs_tag" : null,
  "build_num" : 20,
  "infrastructure_fail" : false,
  "ssh_enabled" : false,
  "committer_email" : "jack@signalfx.com",
  "previous" : {
    "build_num" : 19,
    "status" : "running",
    "build_time_millis" : 0
  },
  "status" : "not_running",
  "committer_name" : "Jack Lindamood",
  "retries" : null,
  "subject" : "Build trigger start",
  "timedout" : false,
  "dont_build" : null,
  "lifecycle" : "not_running",
  "no_dependency_cache" : null,
  "stop_time" : null,
  "build_time_millis" : null,
  "circle_yml" : null,
  "messages" : [ ],
  "is_first_green_build" : false,
  "job_name" : null,
  "start_time" : null,
  "canceler" : null,
  "all_commit_details" : [ {
    "committer_date" : "2015-10-28T19:44:40Z",
    "body" : "a body",
    "author_date" : "2015-10-28T19:44:07Z",
    "committer_email" : "jack@signalfx.com",
    "commit" : "3a6bfed57ea1d93f9f2b332c93786b30f68e7d3e",
    "committer_login" : "cep21",
    "committer_name" : "Jack Lindamood",
    "subject" : "Build trigger start",
    "commit_url" : "https://github.com/signalfx/arepo/commit/3a6bfed57ea1d93f9f2b332c93786b30f68e7d3e",
    "author_login" : "cep21",
    "author_name" : "Jack Lindamood",
    "author_email" : "jack@signalfx.com"
  } ],
  "outcome" : null,
  "vcs_url" : "https://github.com/signalfx/areponame",
  "author_name" : "Jack Lindamood",
  "node" : null,
  "canceled" : false,
  "author_email" : "jack@signalfx.com"
}`

func TestParsingCircleClient(t *testing.T) {
	err := json.Unmarshal([]byte(response), &buildResponse{})
	if err != nil {
		t.Errorf("Cannot parse json: %s", err.Error())
	}
}
