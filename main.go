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

func main() {
	type Item struct {
		Repo    string
		Image   string
		Webhook string
	}

	type Items struct {
		Containers []Item `containers`
	}

	var item Items

	// TODO migrate to a config file/env
	ArtifactoryURI := "http://localhost:8081" // artifactory URI
	// pollTime := 10 // How often to poll the homepage
	filename := "data.yaml"

	fmt.Println(ArtifactoryURI)

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
	// TODO change to loop
	webhook := item.Containers[0].Webhook
	image := item.Containers[0].Image
	repo := item.Containers[0].Repo
	fmt.Println(webhook, image, repo)

	// Create the http client
	client := &http.Client{
		Timeout: time.Second * 3,
	}

	// Perform GET to URI
	// TODO change from the hardcoded repo1/app1
	res, err := client.Get(ArtifactoryURI + "/api/storage/repo1/app1")
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
	// Currently using [0]
	// TODO fix in to a loop
	// The [1:] slices the first letter from realTag, in this case remove /
	realTag := tag.Children[0].URI[1:]
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
