package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NissesSenap/promotionChecker/promoter"
	mdb "github.com/NissesSenap/promotionChecker/repository/hmemdb"
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
	PollTime          int    `pollTime`
	HTTPtimeout       int    `httpTimeout`
}

// Create channel for ctx
var c = make(chan int)

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

	repo := chooseRepo("repo", 3)
	// Create the http client
	// Notice the time.Duration
	client := &http.Client{
		Timeout: time.Second * time.Duration(item.HTTPtimeout),
	}

	// Create base context
	ctx, cancel := context.WithCancel(context.Background())

	// Create the memory db
	service := promoter.NewRedirectService(repo)

	// hmemdbRepo.Store("repo1/app1", "repo1", "app1", []string{"v1.0.0", "v2.0.0"})

	// goroutine start the infinate runner function
	go runner(ctx, &item, client, service)

	// Create a channel that listens for SIGTERM
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-signalCh:
	case <-ctx.Done():
	}

	// Handle potential shutdown like shuting down DB connections
	fmt.Println("*********************************\nShutdown signal received\n*********************************")
	tags, err := service.Read("repo1/app1")
	if err != nil {
		fmt.Println("Unable to find the repoImage")
	}
	fmt.Printf("Here is the tags %v", tags)

	cancel()
}

func runner(ctx context.Context, item *Items, client *http.Client, hmemdbRepo promoter.RedirectRepository) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
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

					// Update db with info
					repoImage := repo + "/" + image

					// TODO remove bellow
					hmemdbRepo.Store(repoImage, repo, image, []string{realTag})
					tags, err := hmemdbRepo.Read(repoImage)
					if err != nil {
						fmt.Println("Unable to find the repoImage")
					}
					fmt.Printf("Here is the tags %v", tags)

					// Update db with info
					/*
						err = promoter.UpdateTags(repoImage, repo, image, []string{realTag}, hmemdbRepo)
						if err != nil {
							log.Fatal("Unable to store things in the memDB")
						}
					*/
				}
			}

			// Sleep for the next pollTime
			time.Sleep(time.Second * time.Duration(item.PollTime))
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

func chooseRepo(tableName string, timeout int) promoter.RedirectRepository {
	switch os.Getenv("URL_DB") {
	case "memDB":
		repo, err := mdb.NewMemDBRepository(tableName, timeout)
		if err != nil {
			log.Fatal(err)
		}
		return repo
	}
	return nil
}
