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

I have tested using **openshift pipelines version 1.2.0**.

In the latest version of tekton there is some **breaking** changes for triggersTemplate

## Test the app in artifactory

If you don't have a JFrog artifactory with pro license sitting around I thought you could use a test subscription.
But it seems like the test subscription doesn't support usage of the rest API just like the open-source version doesn't.

I wrote a bunch of instructions on how to setup artifactory and I don't feel like removing them so you can see them in [Artifactory.md](Artifactory.md)

## testServer for e2e testing

In the mean time I have created a simple testServer looking on how the data looked like in artifactory 6.

Check out [testerServer](testServer/main.go) and run it with `go run main.go` from the testServer folder.

I have also packaged the test server as a container and have a very simple helm chart to be able to run it on k8s.

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
