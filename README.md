# Deathnode
Gracefully Mesos agent destroy system for AWS Autoscaling.

### What's does it do?
Deathnode ensures that the agents from an Autoscaling group in AWS are destroyed only after they have no running tasks, allowing to implement scale-in or red/black deployments with no impact to the customers.

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

### Build
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

### Limitations
Deathnode is still in development, and has not being tested in a real-production environment. 

* Most of the Mesos frameworks doesn't implement Maintenance primitives.
* Maintenance primitives are not available through Mesos native integration
