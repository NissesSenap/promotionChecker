package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	Containers        []Item `containers`
	ArtifactoryURL    string `artifactoryURL`
	ArtifactoryAPIkey string `artifactoryAPIkey`
	ArtifactoryUSER   string `artifactoryUSER`
	pollTime          int
	httpTimeout       int
}

func main() {
	filename := getEnv("CONFIGFILE", "data.yaml")

	var item Items

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

	/* Check if env ARTIFACTORYAPIKEY got a value
	If != "" overwrite the value in item.ArtifactoryAPIkey
	This way env variable allways trumps the config.
	But we can have a config file if we want.
	*/
	apiKEY := getEnv("ARTIFACTORYAPIKEY", "")
	if apiKEY != "" {
		item.ArtifactoryAPIkey = apiKEY
	}

	apiUSER := getEnv("ARTIFACTORYUSER", "")
	if apiUSER != "" {
		item.ArtifactoryUSER = apiUSER
	}

	fmt.Println(item.ArtifactoryURL)

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

		fulURL := item.ArtifactoryURL + "/api/storage/" + repo + "/" + image

		req, _ := http.NewRequest("GET", fulURL, nil)

		// Depending on config use BasicAuth, header or nothing
		if item.ArtifactoryUSER != "" && item.ArtifactoryAPIkey != "" {
			req.SetBasicAuth(item.ArtifactoryUSER, item.ArtifactoryAPIkey)
		} else if item.ArtifactoryAPIkey != "" {
			req.Header.Add("X-JFrog-Art-Api", item.ArtifactoryAPIkey)
		}

		// Perform GET to URI
		res, err := client.Do(req)

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

// getEnv get key environment variable if exist otherwise return defalutValue
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		fmt.Println("In getEnv")
		fmt.Println(value)
		return value
		// TODO add a log debug here and print the value
	}
	return defaultValue
}
