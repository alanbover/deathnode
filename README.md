# Deathnode

[![Build Status](https://travis-ci.org/alanbover/deathnode.svg?branch=master)](https://travis-ci.org/alanbover/deathnode)
[![Go Report Card](https://goreportcard.com/badge/github.com/alanbover/deathnode)](https://goreportcard.com/report/github.com/alanbover/deathnode)
[![Coverage Status](https://coveralls.io/repos/github/alanbover/deathnode/badge.svg?branch=enable_travis_and_improve_doc)](https://coveralls.io/github/alanbover/deathnode?branch=enable_travis_and_improve_doc)

## Purpose
Gracefully Mesos agent destroy system for AWS Autoscaling.

Deathnode ensures that agents from an Autoscaling group in AWS are destroyed only after they have being drained, allowing to implement scale-in or red/black deployments with no impact to the customers.

It's implementation is based using:

* Mesos Maintenance Primitives (http://mesos.apache.org/documentation/latest/maintenance/)
* AWS Autoscaling ProtectFromScaleIn (http://docs.aws.amazon.com/autoscaling/latest/userguide/as-instance-termination.html)

### How does it do it?
Deathnode monitors the autoscaling groups from the Mesos Agents. Whenever it detects that one instance from an autoscaling group should be removed, it will:

*  Find the best agent to be killed
*  Tag the instance as being deleted
*  Remove the instance from the ASG
*  Set the instance in maintenance mode

Then deathnode will keep monitoring this agent, destroying it once it's drained.

## Usage
Here you can find an example of usage:
```
./deathnode -autoscalingGroupName ${ASG_NAME} -delayDelete 300 -mesosUrl ${MESOS_URL} -polling 60 -protectedFrameworks Eremetic -debug
```

## Build
To execute the test, run:
```
make test
```

To build the app, run:
```
make build
```

To build a docker image, run:
```
DOCKERTAG="version" make docker
```

## Limitations
Deathnode is still in development, and test in a real-production environment have just started.

* Most of the Mesos frameworks doesn't implement Maintenance primitives.
* Maintenance primitives are not available through Mesos native integration
