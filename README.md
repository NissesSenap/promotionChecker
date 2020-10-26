# promotionChecker

A simple app to poll a solution like JFrog artifactory over and over again.

Now a days there are webhooks in artifactory but someplace i work don't feel like upgrading nor installing the artifactory webhook plugin.
So lets go old school and poll stuff instead.

I will do some specific coding for artifactory but it shouldn't be hard to adopt another endpoint like nexus.

## Test the app

If you don't have a JFrog artifactory with pro license sitting around you can start a [test subscription](https://jfrog.com/artifactory/start-free/).
Sadly the open-source version of artifactory isn't enough due to the rest API.

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

## TODO

In no particular order.

- Fix the general loops needed
- Refactor to look nicer
- Store all the data in something like a db or file.
- Write a container file
- Create some metrics
- If speed is needed create channels
- Manage SIGTERM
- Use external config files/env
- During the initial start check if there is missing values in the DB.
  - Is it okay that we trigger all the jobs at startup?
  - Should we just make a list of everything currently available and start nothing. This will make it possible to miss triggers.
    - If so we can save the data in a file/in a memory db.
- Create a basic k8s deployment file
- Write a example trigger binding for tekton
- Support artifactory header and manage API key and username
