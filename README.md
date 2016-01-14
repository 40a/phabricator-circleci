# phabricator-circleci [![Circle CI](https://circleci.com/gh/signalfx/phabricator-circleci.svg?style=svg)](https://circleci.com/gh/signalfx/phabricator-circleci) [![Docker Repository on Quay](https://quay.io/repository/signalfx/phabricator-circleci/status "Docker Repository on Quay")](https://quay.io/repository/signalfx/phabricator-circleci)

integration of phabricator and circleci

## How this works

1. Users run ```arc diff```
2. Phabricator uploads their change to the staging area
3. Herald triggers a harbormaster build
  1. Harbormaster does a POST to Lambda, which puts a msg on SQS
4. This code picks up that message
  1. The staging area is moved from a tag to a branch
  2. A build is triggered in CircleCI for this branch
  3. The diff is updated that the build is happening
5. CircleCI finishes the build and executes the notify step
  1. Notify step does a POST to Lambda which puts a msg on SQS
6. This code picks up that message
  1. The staging area branch/tag is deleted
  2. Harbormaster is updated with the build results
  3. The diff is updated with the build results.

## How to run

The easiest way to run this is via the docker image listed above.  It
will require the following env variables when run:

| Variable            | Description  |
|---------------------|------------------------------------------------------|
| SQS_REGION          | The region in AWS that your SQS queue is located in  |
| BUILD_VERBOSE       | If 1, will enable verbose logging                    |
| SQS_QUEUE           | The AWS location of your SQS queue                   |
| BUILD_VERBOSE_FILE  | Where to output your logs                            |
| PHAB_API_TOKEN      | Phabricator API token                                |
| CIRCLECI_TOKEN      | Token to talk to CircleCI                            |

Example env may look like this:

```
'SQS_REGION': 'us-east-1',
'BUILD_VERBOSE': '1',
'SQS_QUEUE': 'https://sqs.us-east-1.amazonaws.com/111111111/some_name',
'BUILD_VERBOSE_FILE': '/var/log/buildtrigger/buildtrigger.log.json',
'PHAB_API_TOKEN': 'api-XYZYOUROTKENHERE',
'CIRCLECI_TOKEN': '1312321XYZYOURTOKENHERE',
```

## Configure Phabricator to trigger the build

### Configure harbormaster to understand builds

We use SQS as our way to communicate between phabricator and this circleci
integration.  To enable this communication, setup a harbormaster build step
in phabricator.  It should post to a URL that can accept the build trigger
and should contain in the URL information similar to the following:

```
https://xyz.execute-api.us-east-1.amazonaws.com/prod/xyzabc?phid=${target.phid}&diff=${buildable.diff}&revision=${buildable.revision}&staging_ref=${repository.staging.ref}&staging_uri=${repository.staging.uri}&callsign=${repository.callsign}
```

### Configure Herald to trigger a build

Herald will then trigger the build on a push.  Inside herald, create a
condition that runs the build plan setup above whenever the repository
is any repository you want to watch.

### Configure Diffusion to use a staging area

Inside phabricator's diffusion, setup a staging area for youru
application.  I generally have all applications share the same staging area.

## Configure AWS lambda to store a SQS message

To do this create a lambda function similar to the following:

```
import json
import boto3

def lambda_handler(event, context):
    msg = json.dumps(event)
    sqs = boto3.resource('sqs')
    queue = sqs.get_queue_by_name(QueueName='your_queue_name')
    response = queue.send_message(MessageBody=msg)
    return response
```

You'll want to expose this on an API endpoint.  This API endpoint will
be used by phabricator to trigger builds as well as CircleCI to signal
a build is done.

## Configure circle.yml to notify SQS (via lambda) when a build is done

This will be a final notify hook in your circle.yml file like the following:

```
notify:
  webhooks:
    - url: https://xyzabcdefg.execute-api.us-east-1.amazonaws.com/prod/xyzabc
```
