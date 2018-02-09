# Deathnode

[![Build Status](https://travis-ci.org/alanbover/deathnode.svg?branch=master)](https://travis-ci.org/alanbover/deathnode)
[![Go Report Card](https://goreportcard.com/badge/github.com/alanbover/deathnode)](https://goreportcard.com/report/github.com/alanbover/deathnode)
[![Coverage Status](https://coveralls.io/repos/github/alanbover/deathnode/badge.svg?branch=master)](https://coveralls.io/github/alanbover/deathnode?branch=master)

## Purpose
Gracefully Mesos agent destroy system for AWS Autoscaling.

Deathnode ensures that agents from an Autoscaling group in AWS are destroyed only after they have being drained, allowing to implement scale-in or red/black deployments with no impact to the customers.

It's implementation is based using:

* Mesos Maintenance Primitives (http://mesos.apache.org/documentation/latest/maintenance/)
* AWS Autoscaling ProtectFromScaleIn (http://docs.aws.amazon.com/autoscaling/latest/userguide/as-instance-termination.html)
* AWS Autoscaling Lifecycle Hooks (http://docs.aws.amazon.com/autoscaling/latest/userguide/lifecycle-hooks.html)

### How does it do it?
Deathnode monitors the autoscaling groups from the Mesos Agents. Whenever it detects that one instance from an autoscaling group should be removed, it will:

*  Find the best agent to be killed
*  Tag the instance as being deleted
*  Remove instance protection from the instance
*  Set the instance in maintenance mode

Then deathnode will keep monitoring this agent, completing destroy lifecycle once it's drained.

## Usage
Here you can find an example of usage:
```
./deathnode -autoscalingGroupName ${ASG_NAME} -delayDelete 300 -mesosUrl ${MESOS_URL} -polling 60 -protectedFrameworks Eremetic -debug
```

### Constraints
When removing an instance, contraints are used by deathnode to filter which instances are not able to be picked up as candidates (best efford). Multiple contraints can be specified.

* noContraint: Applies no constraints
* protectedConstraint: Do not pick instances that has tasks from protected frameworks
* filterFrameworkConstraint: Do not pick instances that has tasks from the specified framework
* taskNameRegexpConstraint: Do not pick instances that has tasks that it's name match a certain regexp

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
* Most of the Mesos frameworks doesn't implement Maintenance primitives.
* Maintenance primitives are not available through Mesos native integration
