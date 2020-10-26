package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/yaml.v2"
)

type Tags struct {
	Repo         string
	Path         string
	Created      string
	CreatedBy    string
	LastModified string
	ModifiedBy   string
	LastUpdated  string
	Children     []Children
	URI          string
}

type Children struct {
	URI    string
	Folder bool
}

type Item struct {
	Repo    string
	Image   string
	Webhook string
}

type Items struct {
	Containers     []Item `containers`
	ArtifactoryURL string `artifactoryURL`
	pollTime       int
	httpTimeout    int
}

func main() {
	var item Items

	filename := "data.yaml"

	fmt.Println(item.ArtifactoryURL)

	// Read the config file
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	// unmarshal the data
	err = yaml.Unmarshal(source, &item)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("--- config:\n%v\n\n", item)

	// Create the http client
	// Notice the time.Duration
	client := &http.Client{
		Timeout: time.Second * time.Duration(item.httpTimeout),
	}
	runner(&item, client)
}

func runner(item *Items, client *http.Client) {
	for i := range item.Containers {

		webhook := item.Containers[i].Webhook
		image := item.Containers[i].Image
		repo := item.Containers[i].Repo
		fmt.Println(webhook, image, repo)

		// Perform GET to URI
		res, err := client.Get(item.ArtifactoryURL + "/api/storage/" + repo + "/" + image)
		if err != nil {
			fmt.Println(err)
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}

		tag := Tags{}

		// Unmarshal the data
		jsonErr := json.Unmarshal(body, &tag)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}

		// Go through all the tags
		// TODO add a comparision if these tags already exist
		for f := range tag.Children {
			// The [1:] slices the first letter from realTag, in this case remove /
			realTag := tag.Children[f].URI[1:]
			fmt.Println(realTag)

			// Post to the webhook endpoint
			webhookValues := map[string]string{"image": image, "repo": repo, "tag": realTag}
			jsonValue, err := json.Marshal(webhookValues)
			if err != nil {
				fmt.Println(err)
			}

			_, err = client.Post(webhook, "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}
