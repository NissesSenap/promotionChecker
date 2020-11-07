# promotionChecker

A simple app to poll a solution like JFrog artifactory over and over again.

Now a days there are webhooks in artifactory but someplace i work don't feel like upgrading nor installing the artifactory webhook plugin.
So lets go old school and poll stuff instead.

I will do some specific coding for artifactory but it shouldn't be hard to adopt another endpoint like nexus.

## Tekton webhook

I'm using tekton to trigger pipelines and thats why I need info when a new image tag is in artifactory.

I have created a example tekton CEL binding listening for **tag** and **image**.

To use it first you need to install the tekton operator, ether [follow](https://tekton.dev/docs/triggers/install/)
or if you are using OCP you can search for `tekton` in OLM and install it from there.

To deploy the helm chart ether use the make commands or use helm install under deploy/tekton-example.

I have tested using **openshift pipelines version 1.1.1**.

In the latest version of tekton there is some **breaking** changes for triggersTemplate

## Test manually

The default values in the values is set to work with the following config together with the test server.

```bash
oc new-project promotion
make test/helm
make tekton/helm
make helm

# Assuming that your route is name test-promotion-test-promotion-checker
# This will trigger a change in the tags on the test server and should trigger a webhook which should trigger the trigger-template
curl -k https://$(oc get route test-promotion-test-promotion-checker -o go-template --template='{{.spec.host}}')/update

```

If you look at the logs of the task that ran you should see something like:

app1 & MyNewTAG

## Test the app artifactory

If you don't have a JFrog artifactory with pro license sitting around you can start a [test subscription](https://jfrog.com/artifactory/start-free/).
Sadly the open-source version of artifactory isn't enough due to the rest API.

### Issues with Artifactory rest API

Don't know if it's due to test license or if it's due to only using a pro license but I can't get any rest API queries to work.
I have created a JIRA ticket with Artifactory about this to get a clarification.

In the mean time I have created a simple testServer looking on how the data looked like in artifactory 6.something.

Check out [testerServer](testServer/main.go) and run it with `go run main.go` from the testServer folder.

### Start artifactory

For instructions on how to start artifactory see [here](https://jfrog.com/artifactory/install/)

I use podman

```shell
podman volume create artifactory-data
podman pull docker.bintray.io/jfrog/artifactory-pro:latest
podman run -d --name artifactory -p 8082:8082 -p 8081:8081 -v artifactory-data:/var/opt/jfrog/artifactory docker.bintray.io/jfrog/artifactory-pro:latest

```

Go to localhost:8082

The default username & password:

```shell
username: admin
password: password
```

### Configure artifactory

Follow the instructions when you first login and add your license information that you should have gotten in a email.

- Create a repo called repo1
- Add a user, to make life easy create a admin user, I called mine user1.

Lets upload a container so we have something to test against.

```shell
podman login --tls-verify=false -u user1 localhost:8082
# Type your password

# Push image to repo1 registry app1 image with tag latest
podman push --tls-verify=false localhost:8082/repo1/app1:latest
```

#### Generate artifactory API Key

You need to setup a API Key for user1

Go in to Profile -> verify your password -> Generate API key

It should look something like this:

AKCp8hyiw8VvXmS8jiy.......................kgs4WoVAaLsA5Up2v5B

#### Test API key

I store my API key in a variable called API key for easy usage.

Seems like there is a issue with ether the latest version of artifactory or I can't use the rest API using a Trial license
Testing on: Trial license 7.10.2 rev 71002900

```bash
export APIKEY=AKCp8hyiw8VvXmS8jiy.......................kgs4WoVAaLsA5Up2v5B

curl -H "X-JFrog-Art-Api:$APIKEY" -X GET http://localhost:8081/access/api/v1/system/ping

curl -H "X-JFrog-Art-Api:$APIKEY" -X GET http://localhost:8081/api/docker/repo1/v2/_catalog

curl -H "X-JFrog-Art-Api:$APIKEY" -X GET http://localhost:8081/api/storage/repo1/

curl -H "X-JFrog-Art-Api:$APIKEY" -X GET http://localhost:8081/api/storage/repo1/app1/

curl -H "X-JFrog-Art-Api:$APIKEY" -X GET http://localhost:8081/api/docker/repo1/v2/app1/tags/list

```

## Assumptions

- Currently assume that it's okay that I have missed updates in the repos
  - At startup create a status of the current env and save in the DB.
  - From there start creating webhooks depending on new changes to the repos

## TODO

In no particular order.

- Refactor to look nicer
- Create some metrics
- If speed is needed create channels to perform the API requests
- Improve the helm config, with more config logic
- Add liveliness for k8s
- Two http clients, one might want to enforce https while the other doesn't.
  Might be easier to overwrite config depending on needs...
  - One for artifactory
  - One for talking to the webhook
- Break out secrets in to a separate k8s file or use env variables
- Write units tests
- Use testServer for simple e2e tests
