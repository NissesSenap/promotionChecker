# testServer

Something something JFrog artifactory rest API doesn't seem to work with
a test license.

Since I only need one simple get commands I spin up a server responding with the json I got from my prod instance.
Far from the best code but it dose the job.

I created a bug explaining  the issue in the JFrog [Jira](https://www.jfrog.com/jira/browse/RTFACT-23754)

## Update

To verify that it works to see a new tag and my DB will behave I created a super ugly way to update it without a DB in the testServer.
Just curl /update and it will add a new tag to the list.

`curl localhost:8081/update`

Or if you are using the default settings for helm in the project promotion.

`curl -k https://test-promotion-test-promotion-checker-promotion.apps-crc.testing/update`

## /tags

Use a GET
/api/storage/repo1/app1 & /api/storage/repo2/app2
The will return different payload

## /webhook

Tekton [trigger](https://github.com/tektoncd/triggers/tree/master/examples)
Use a POST that look something like this:

```bash
curl -X POST \
  http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -H 'X-Hub-Signature: sha1=2da37dcb9404ff17b714ee7a505c384758ddeb7b' \
  -d '{
	"repository":
	{
		"url": "https://github.com/tektoncd/triggers.git"
	}
}'
```
