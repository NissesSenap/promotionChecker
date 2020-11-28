package promoter

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

const ignoreTag string = "/_uploads"

type config struct {
	image     string
	repo      string
	repoImage string
	webhook   string
	item      *Items
	client    *http.Client
	service   RedirectRepository
}

func newConfig(image string, repo string, webhook string, repoImage string, item *Items, client *http.Client, service RedirectRepository) *config {
	return &config{
		image:     image,
		repo:      repo,
		webhook:   webhook,
		repoImage: repoImage,
		item:      item,
		client:    client,
		service:   service,
	}

}

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
				repoImage := repo + "/" + image
				config := newConfig(image, repo, webhook, repoImage, item, client, service)
				zap.S().Infof("Config to check webhook %s, image: %s, repo: %s: ", webhook, image, repo)

				tag, err := config.requestArtData()
				if err != nil {
					zap.S().DPanic("Unable to get data from artifactory: ", err)
					os.Exit(1)
				}

				// Go through all the tags
				for f := range tag.Children {

					err = config.runner(tag.Children[f].URI)
					if err != nil {
						zap.S().DPanic(err)
						os.Exit(1)
					}

				}
			}

			// Sleep for the next pollTime
			time.Sleep(time.Second * time.Duration(item.PollTime))
		}
	}
}

// Runner the main for loop of promoterChecker
func (c *config) runner(tag string) error {
	// Shorter to write realTag then tag.Children[f].URI
	//realTag := tag.Children[f].URI

	// Check the current tags
	existingTags, err := c.service.Read(c.repoImage)
	if err != nil {
		return err
	}

	// Returns true if a we have gotten a new tag
	//  and the new tag doesn't contain /_uploads
	if tag != ignoreTag && StringNotInSlice(tag, existingTags) {
		zap.S().Infof("Got a new tag in the image: %s ,repo: %s, newTag %v", c.image, c.repo, tag)

		err = c.webhookPOST(tag)
		if err != nil {
			return err
		}

		// Update db with info
		err = c.service.UpdateTags(c.repoImage, c.repo, c.image, []string{tag})
		if err != nil {
			return err
		}

		NrTagsPromoted.Inc()
		// Verify the existing tags
		// TODO add a if to check if in debug, there is no need to run this all the time
		tags, err := c.service.Read(c.repoImage)
		if err != nil {
			return err
		}
		zap.S().Debug("Current tags in the DB: ", tags)

	}
	return nil
}

func (c *config) webhookPOST(tag string) error {
	// Post to the webhook endpoint
	// Notice the slice of realTag, removing the / that is stored in the DB.
	webhookValues := map[string]string{"image": c.image, "repo": c.repo, "tag": tag[1:]}
	jsonValue, err := json.Marshal(webhookValues)
	if err != nil {
		zap.S().Error(err)
	}

	req, err := http.NewRequest("POST", c.webhook, bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	// Adding headers to the webhook request
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Event-Promoter-Checker-Com", "webhook")

	// Add a secret in the webhook so you can verify it in the EventListener
	req.Header.Add("X-Secret-Token", c.item.WebhookSecret)

	start := time.Now()
	res, err := c.client.Do(req)

	if err != nil {
		return err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr
	}

	duration := time.Since(start)
	HistWebhook.WithLabelValues(c.repoImage).Observe(duration.Seconds())

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
		config := newConfig(image, repo, webhook, "", item, client, service)
		zap.S().Debugf("Config to check: ", webhook, image, repo)

		tag, err := config.requestArtData()
		if err != nil {
			return err
		}
		// Store all the tags in one slice
		var slicedTags []string
		for s := range tag.Children {
			realTag := tag.Children[s].URI
			slicedTags = append(slicedTags, realTag)
		}

		repoImage := repo + "/" + image

		// Store all the existing tags in the memDB
		err = service.Store(repoImage, repo, image, slicedTags)
		if err != nil {
			return err
		}
	}
	return nil
}

// requestArtData talks to repo storage on a specific endpoints and check what tags exist
func (c *config) requestArtData() (*Tags, error) {
	fulURL := c.item.ArtifactoryURL + "/api/storage/" + c.repo + "/" + c.image

	req, err := http.NewRequest("GET", fulURL, nil)
	if err != nil {
		return nil, err
	}

	// Depending on config use BasicAuth, header or nothing
	if c.item.ArtifactoryUSER != "" && c.item.ArtifactoryAPIkey != "" {
		req.SetBasicAuth(c.item.ArtifactoryUSER, c.item.ArtifactoryAPIkey)
	} else if c.item.ArtifactoryAPIkey != "" {
		req.Header.Add("X-JFrog-Art-Api", c.item.ArtifactoryAPIkey)
	}

	// histogram timer start
	start := time.Now()

	// Perform GET to URI
	res, err := c.client.Do(req)

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
	HistArtifactory.WithLabelValues(c.repo + c.image).Observe(duration.Seconds())

	return &tag, nil
}
