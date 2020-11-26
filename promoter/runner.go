package promoter

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const ignoreTag string = "/_uploads"

// TODO create method to throw around all the values.
// TODO remove a bunch of logs and just return error instead.

// MainRunner the main for loop of promoterChecker
func MainRunner(ctx context.Context, item *Items, client *http.Client, service RedirectRepository) {
	select {
	case <-ctx.Done():
		return
	default:
		for {
			for i := range item.Containers {

				webhook := item.Containers[i].Webhook
				image := item.Containers[i].Image
				repo := item.Containers[i].Repo
				zap.S().Infof("Config to check webhook %s, image: %s, repo: %s: ", webhook, image, repo)

				tag, err := requestArtData(image, repo, item, client)
				if err != nil {
					zap.S().DPanic("Unable to get data from artifactory: ", err)
				}

				// Go through all the tags
				for f := range tag.Children {

					err = Runner(tag.Children[f].URI, image, repo, webhook, item, client, service)
					// TODO still needs better error handeling, should i panic here? Should probably do it later...
					if err != nil {
						panic(err)
					}

				}
			}

			// Sleep for the next pollTime
			time.Sleep(time.Second * time.Duration(item.PollTime))
		}
	}
}

// Runner the main for loop of promoterChecker
func Runner(tag string, image string, repo string, webhook string, item *Items, client *http.Client, service RedirectRepository) error {
	repoImage := repo + "/" + image

	// Shorter to write realTag then tag.Children[f].URI
	//realTag := tag.Children[f].URI

	// Check the current tags
	existingTags, err := service.Read(repoImage)
	if err != nil {
		zap.S().Panic("Unable to find the repoImage")
		return err
	}

	// Returns true if a we have gotten a new tag
	//  and the new tag doesn't contain /_uploads
	if tag != ignoreTag && StringNotInSlice(tag, existingTags) {
		zap.S().Infof("Got a new tag in the image: %s ,repo: %s, newTag %v", image, repo, tag)

		err = webhookPOST(tag, image, repo, webhook, repoImage, item, client, service)
		if err != nil {
			return err
		}

		// Update db with info
		err = service.UpdateTags(repoImage, repo, image, []string{tag})
		if err != nil {
			zap.S().Error(err)
			return err
		}

		NrTagsPromoted.Inc()
		// Verify the existing tags
		// TODO add a if to check if in debug, there is no need to run this all the time
		tags, err := service.Read(repoImage)
		if err != nil {
			zap.S().Panic("Unable to find the repoImage", err)
			return err
		}
		zap.S().Debug("Current tags in the DB: ", tags)

	}
	return nil
}

func webhookPOST(tag string, image string, repo string, webhook string, repoImage string, item *Items, client *http.Client, service RedirectRepository) error {
	// Post to the webhook endpoint
	// Notice the slice of realTag, removing the / that is stored in the DB.
	webhookValues := map[string]string{"image": image, "repo": repo, "tag": tag[1:]}
	jsonValue, err := json.Marshal(webhookValues)
	if err != nil {
		zap.S().Error(err)
	}

	req, err := http.NewRequest("POST", webhook, bytes.NewBuffer(jsonValue))
	if err != nil {
		zap.S().Panic("Unable to post the webhook: ", err)
		return err
	}

	// Adding headers to the webhook request
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Event-Promoter-Checker-Com", "webhook")

	// Add a secret in the webhook so you can verify it in the EventListener
	req.Header.Add("X-Secret-Token", item.WebhookSecret)

	start := time.Now()
	res, err := client.Do(req)

	if err != nil {
		return err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		zap.S().Error(readErr)
		return readErr
	}

	duration := time.Since(start)
	HistWebhook.WithLabelValues(repoImage).Observe(duration.Seconds())

	// No need to do a marshall stuff. Just a pain to maintain the different webhooks, just want output for logs.
	zap.S().Infof("Output from webhook: %s", string(body))

	return nil
}

// InitialRunner creates the initial data in the database, getting all the data that currently exist in your repo
func InitialRunner(item *Items, client *http.Client, service RedirectRepository) error {
	for i := range item.Containers {

		webhook := item.Containers[i].Webhook
		image := item.Containers[i].Image
		repo := item.Containers[i].Repo
		zap.S().Debugf("Config to check: ", webhook, image, repo)

		tag, err := requestArtData(image, repo, item, client)
		if err != nil {
			zap.S().DPanic("Unable to get data from artifactory: ", err)
		}
		// Store all the tags in one slice
		var slicedTags []string
		for s := range tag.Children {
			realTag := tag.Children[s].URI
			slicedTags = append(slicedTags, realTag)
		}

		repoImage := repo + "/" + image

		// Store all the existing tags in the memDBgolangci-lint
		err = service.Store(repoImage, repo, image, slicedTags)
		if err != nil {
			zap.S().DPanic("Unable to store our data")
			return err
		}
	}
	return nil
}

// requestArtData talks to repo storage on a specific endpoints and check what tags exist
func requestArtData(image string, repo string, item *Items, client *http.Client) (*Tags, error) {
	fulURL := item.ArtifactoryURL + "/api/storage/" + repo + "/" + image

	req, err := http.NewRequest("GET", fulURL, nil)
	if err != nil {
		return nil, err
	}

	// Depending on config use BasicAuth, header or nothing
	if item.ArtifactoryUSER != "" && item.ArtifactoryAPIkey != "" {
		req.SetBasicAuth(item.ArtifactoryUSER, item.ArtifactoryAPIkey)
	} else if item.ArtifactoryAPIkey != "" {
		req.Header.Add("X-JFrog-Art-Api", item.ArtifactoryAPIkey)
	}

	// histogram timer start
	start := time.Now()

	// Perform GET to URI
	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, err
	}

	tag := Tags{}

	// Unmarshal the data
	jsonErr := json.Unmarshal(body, &tag)
	if jsonErr != nil {
		return nil, err
	}

	// calculate the duration since the timer started & add to the histogram
	duration := time.Since(start)
	HistArtifactory.WithLabelValues(repo + image).Observe(duration.Seconds())

	return &tag, nil
}
