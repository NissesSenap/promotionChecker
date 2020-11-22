package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/NissesSenap/promotionChecker/build"
	"github.com/NissesSenap/promotionChecker/promoter"
	mdb "github.com/NissesSenap/promotionChecker/repository/hmemdb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

// Tags artifactory output from rest call
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

// Children artifactory output for all tags
type Children struct {
	URI    string
	Folder bool
}

// Item the config values that the app check
type Item struct {
	Repo    string
	Image   string
	Webhook string
}

// Items config file struct
type Items struct {
	Containers        []Item `yaml:"containers"`
	ArtifactoryURL    string `yaml:"artifactoryURL"`
	ArtifactoryAPIkey string `yaml:"artifactoryAPIkey"`
	ArtifactoryUSER   string `yaml:"artifactoryUSER"`
	PollTime          int    `yaml:"pollTime"`
	HTTPtimeout       int    `yaml:"httpTimeout"`
	HTTPinsecure      bool   `yaml:"httpInsecure"`
	WebhookSecret     string `yaml:"webhookSecret"`
	DBType            string `yaml:"dbType"`
	EndpointPort      int    `yaml:"endpointPort"`
}

func initZapLog() *zap.Logger {
	//config := zap.NewDevelopmentConfig()
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := config.Build()
	return logger
}

const ignoreTag = "/_uploads"

func main() {
	loggerMgr := initZapLog()
	zap.ReplaceGlobals(loggerMgr)
	defer loggerMgr.Sync() // flushes buffer, if any
	logger := loggerMgr.Sugar()

	logger.Infof("promotionChecker Version: %s, BuildDate: %s", build.Version, build.BuildDate)

	filename := getEnv("CONFIGFILE", "data.yaml")

	var item Items

	// Read the config file
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.DPanic(err)
	}

	// unmarshal the data
	err = yaml.Unmarshal(source, &item)
	if err != nil {
		logger.Fatalf("error: %v", err)
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

	logger.Info("ArtifactoryURL: ", item.ArtifactoryURL)

	logger.Infof("config: \n%v", item)

	/* Choose what kind of database that we should use
	Currently only memDB is supported
	*/
	// TODO perfrom a ENUM check on item.DBType
	repo := chooseRepo("repo", 3, item.DBType)

	// Allow to use https insecurely
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: item.HTTPinsecure},
	}

	// Create the http client
	// Notice the time.Duration
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * time.Duration(item.HTTPtimeout),
	}

	// Create base context
	ctx, cancel := context.WithCancel(context.Background())

	// Create the memory db
	service := promoter.NewRedirectService(repo)

	// Start the initial function that adds the current info to the memDB
	err = initialRunner(&item, client, service)
	if err != nil {
		// TODO need to decide how to manage panics and actually panic. Should probably use the singalCh ^^
		os.Exit(1)

	}

	// goroutine start the infinate runner function
	go runner(ctx, &item, client, service)

	// Starting metrics http server & endpoint
	http.Handle("/metrics", promhttp.Handler())
	err = http.ListenAndServe(":"+strconv.Itoa(item.EndpointPort), nil)
	if err != nil {
		logger.Panic("Unable to start http server")
	}

	// Create a channel that listens for SIGTERM
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-signalCh:
	case <-ctx.Done():
	}

	// Handle potential shutdown like shuting down DB connections
	logger.Info("Shutdown signal received")
	// TODO rewrite to show all the tags in the DB, add a if statment checking for level only if debug mode do this
	tags, err := service.Read("repo1/app1")
	if err != nil {
		logger.Errorf("Unable to get data from memDB", err)
	}
	logger.Debugf("This is the tags found in repo1/app1:", tags)

	cancel()
}

func runner(ctx context.Context, item *Items, client *http.Client, hmemdbRepo promoter.RedirectRepository) {
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

				tag, err := requestArtData(webhook, image, repo, item, client)
				if err != nil {
					zap.S().DPanic("Unable to get data from artifactory: ", err)
				}

				// Go through all the tags
				for f := range tag.Children {
					repoImage := repo + "/" + image

					// Shorter to write realTag then tag.Children[f].URI
					realTag := tag.Children[f].URI

					// Check the current tags
					existingTags, err := hmemdbRepo.Read(repoImage)
					if err != nil {
						zap.S().Panic("Unable to find the repoImage")
					}

					// Returns true if a we have gotten a new tag
					//  and the new tag doesn't contain /_uploads
					if realTag != ignoreTag && promoter.StringNotInSlice(realTag, existingTags) {
						zap.S().Infof("Got a new tag in the image: %s ,repo: %s, newTag %v", image, repo, realTag)

						// Post to the webhook endpoint
						// Notice the slice of realTag, removing the / that is stored in the DB.
						webhookValues := map[string]string{"image": image, "repo": repo, "tag": realTag[1:]}
						jsonValue, err := json.Marshal(webhookValues)
						if err != nil {
							zap.S().Error(err)
						}

						req, err := http.NewRequest("POST", webhook, bytes.NewBuffer(jsonValue))
						if err != nil {
							zap.S().Panic("Unable to post the webhook: ", err)
							return
						}

						// Adding headers to the webhook request
						req.Header.Set("Content-Type", "application/json")
						req.Header.Add("Event-Promoter-Checker-Com", "webhook")

						// Add a secret in the webhook so you can verify it in the EventListener
						req.Header.Add("X-Secret-Token", item.WebhookSecret)

						start := time.Now()
						res, err := client.Do(req)

						if err != nil {
							return
						}

						if res.Body != nil {
							defer res.Body.Close()
						}

						body, readErr := ioutil.ReadAll(res.Body)
						if readErr != nil {
							zap.S().Error(readErr)
						}

						duration := time.Since(start)
						promoter.HistWebhook.WithLabelValues(repoImage).Observe(duration.Seconds())

						// No need to do a marshall stuff. Just a pain to maintain the different webhooks, just want output for logs.
						zap.S().Infof("Output from webhook: %s", string(body))

						// Update db with info

						err = hmemdbRepo.UpdateTags(repoImage, repo, image, []string{realTag})
						if err != nil {
							zap.S().Error(err)
						}

						promoter.NrTagsPromoted.Inc()
						// Verify the existing tags
						// TODO add a if to check if in debug, there is no need to run this all the time
						tags, err := hmemdbRepo.Read(repoImage)
						if err != nil {
							zap.S().Panic("Unable to find the repoImage", err)
						}
						zap.S().Debug("Current ags in the DB: ", tags)

					}
				}
			}

			// Sleep for the next pollTime
			time.Sleep(time.Second * time.Duration(item.PollTime))
		}
	}
}

func initialRunner(item *Items, client *http.Client, hmemdbRepo promoter.RedirectRepository) error {
	for i := range item.Containers {

		webhook := item.Containers[i].Webhook
		image := item.Containers[i].Image
		repo := item.Containers[i].Repo
		zap.S().Debugf("Config to check: ", webhook, image, repo)

		tag, err := requestArtData(webhook, image, repo, item, client)
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

		// Store all the existing tags in the memDB
		// TODO I'm calling the hmemdbRepo directly... I shouldn't do that.
		err = hmemdbRepo.Store(repoImage, repo, image, slicedTags)
		if err != nil {
			zap.S().DPanic("Unable to store our data")
			return err
		}
	}
	return nil
}

func requestArtData(webhook string, image string, repo string, item *Items, client *http.Client) (*Tags, error) {
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
	promoter.HistArtifactory.WithLabelValues(repo + image).Observe(duration.Seconds())

	return &tag, nil
}

// getEnv get key environment variable if exist otherwise return defalutValue
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		zap.S().Info(value)

		return value
	}
	return defaultValue
}

func chooseRepo(tableName string, timeout int, dbType string) promoter.RedirectRepository {
	switch dbType {
	case "memDB":
		repo, err := mdb.NewMemDBRepository(tableName, timeout)
		if err != nil {
			zap.S().Fatal(err)
		}
		return repo
	}
	return nil
}
