package main

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

var phabResp1 = `{
    "error_code": null,
    "error_info": null,
    "result": {
        "12961": {
            "authorEmail": "jack@signalfx.com",
            "authorName": "Jack Lindamood",
            "bookmark": null,
            "branch": "ddagent",
            "changes": [
                {
                    "addLines": "29",
                    "awayPaths": [],
                    "commitHash": null,
                    "currentPath": "base/thing.conf",
                    "delLines": "0",
                    "fileType": "1",
                    "hunks": [
                        {
                            "addLines": null,
                            "corpus": "stuff",
                            "delLines": null,
                            "isMissingNewNewline": null,
                            "isMissingOldNewline": null,
                            "newLength": "29",
                            "newOffset": "1",
                            "oldLength": "0",
                            "oldOffset": "0"
                        }
                    ],
                    "id": "119706",
                    "metadata": {
                        "line:first": 1
                    },
                    "newProperties": {
                        "unix:filemode": "100644"
                    },
                    "oldPath": null,
                    "oldProperties": [],
                    "type": "1"
                }
            ],
            "creationMethod": "arc",
            "dateCreated": "1444067397",
            "dateModified": "1446083066",
            "description": "This is a test message",
            "id": "12961",
            "lintStatus": "0",
            "properties": {
                "local:commits": {
                    "58d29ddf4afbf6a3e33b34dfc973ff31ca01e544": {
                        "author": "Jack Lindamood",
                        "authorEmail": "jack@signalfx.com",
                        "commit": "58d29ddf4afbf6a3e33b34dfc973ff31ca01e544",
                        "message": "add stuff",
                        "parents": [
                            "d850c543770e8ec17754c40e4d1363e118ab0bc9"
                        ],
                        "summary": "Add setup",
                        "time": "1443640773",
                        "tree": "4eb041a1bba0ae2597c74001d1ad55054fcfb873"
                    }
                }
            },
            "revisionID": "6848",
            "sourceControlBaseRevision": "d850c543770e8ec17754c40e4d1363e118ab0bc9",
            "sourceControlPath": null,
            "sourceControlSystem": "git",
            "unitStatus": "0"
        }
    }
}
`

func TestDecode(t *testing.T) {
	Convey("Decoding conduit responses", t, func() {
		r := queryResult{}
		So(json.Unmarshal([]byte(phabResp1), &r), ShouldBeNil)
		So(r.Result["12961"].ID, ShouldEqual, "12961")
		So(r.Result["12961"].RevisionID, ShouldEqual, "6848")
	})

}
