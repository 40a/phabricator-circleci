# phabricator-circleci [![Circle CI](https://circleci.com/gh/signalfx/phabricator-circleci.svg?style=svg)](https://circleci.com/gh/signalfx/phabricator-circleci) [![Docker Repository on Quay](https://quay.io/repository/signalfx/phabricator-circleci/status "Docker Repository on Quay")](https://quay.io/repository/signalfx/phabricator-circleci)

integration of phabricator and circleci

## How this works

1. Users run ```arc diff```
1. Phabricator uploads their change to the staging area
1. Herald triggers a harbormaster build
   1. Harbormaster does a POST to Lambda, which puts a msg on SQS
1. This code picks up that message
   1. The staging area is moved from a tag to a branch
   1. A build is triggered in CircleCI for this branch
   1. The diff is updated that the build is happening
1. CircleCI finishes the build and executes the notify step
   1. Notify step does a POST to Lambda which puts a msg on SQS
1. This code picks up that message
   1. The staging area branch/tag is deleted
   1. Harbormaster is updated with the build results
   1. The diff is updated with the build results.

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
| PHAB_URL            | URL of phabricator to post build results             |

Example env may look like this:

```
'SQS_REGION': 'us-east-1',
'BUILD_VERBOSE': '1',
'SQS_QUEUE': 'https://sqs.us-east-1.amazonaws.com/111111111/some_name',
'BUILD_VERBOSE_FILE': '/var/log/buildtrigger/buildtrigger.log.json',
'PHAB_API_TOKEN': 'api-XYZYOUROTKENHERE',
'PHAB_URL': 'http://myphab.mycompany.org',
'CIRCLECI_TOKEN': '1312321XYZYOURTOKENHERE',
```

This code will also attempt to clean up branches in the phabricator staging
area that are no longer needed.  To do this, it will probably need SSH access
to the staging area git repository.  We do this by cross mounting a /root/.ssh
for this docker image with a directory that contains a SSH key that allows
read/write access to only our staging area.

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

### Allow docker image to push to staging repository

To allow this docker image to push to a staging repository, you can cross mount
/root/.ssh inside the docker image to some directory on your running server that
has a ssh key that can push to the staging repository.

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

I expose this endpoint via the API gateway. Forward the query strings
`diff`, `revision`, `auth`, and `phid`.  And map the content types
`application/json` and `application/x-www-form-urlencoded` to code like
the following:

```
#set($params = $input.params())
{
    "formparams" : $input.json('$'),
    "allParamsJson" : {
    #foreach($type in $params.keySet())
    "$type" : {
            #foreach($paramName in $params.get($type).keySet())
                "$paramName" : "$util.escapeJavaScript($params.get($type).get($paramName))"
                #if($foreach.hasNext),
                #end
            #end
        }
    #if($foreach.hasNext),
    #end
    #end
    }
}
```

## Configure circle.yml to notify SQS (via lambda) when a build is done

This will be a final notify hook in your circle.yml file like the following:

```
notify:
  webhooks:
    - url: https://xyzabcdefg.execute-api.us-east-1.amazonaws.com/prod/xyzabc
```
