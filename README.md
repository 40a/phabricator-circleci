# phabricator-circleci [![Circle CI](https://circleci.com/gh/signalfx/phabricator-circleci.svg?style=svg)](https://circleci.com/gh/signalfx/phabricator-circleci) [![Docker Repository on Quay](https://quay.io/repository/signalfx/phabricator-circleci/status "Docker Repository on Quay")](https://quay.io/repository/signalfx/phabricator-circleci)

integration of phabricator and circleci

# How to run

The easiest way to run this is via the docker image listed above.  It
will require the following env variables when run:

| Variable  | Description  |
|---|---|
| SQS_REGION  | The region in AWS that your SQS queue is located in  |
| BUILD_VERBOSE  | If 1, will enable verbose logging  |
| SQS_QUEUE  | The AWS location of your SQS queue  |
| BUILD_VERBOSE_FILE  | Where to output your logs  |
| PHAB_API_TOKEN  | Phabricator API token  |
| CIRCLEDI_TOKEN  | Token to talk to CircleCI  |

Example env may look like this:

```
'SQS_REGION': 'us-east-1',
'BUILD_VERBOSE': '1',
'SQS_QUEUE': 'https://sqs.us-east-1.amazonaws.com/111111111/some_name',
'BUILD_VERBOSE_FILE': '/var/log/buildtrigger/buildtrigger.log.json',
'PHAB_API_TOKEN': 'api-XYZYOUROTKENHERE',
'CIRCLEDI_TOKEN': '1312321XYZYOURTOKENHERE',
```
