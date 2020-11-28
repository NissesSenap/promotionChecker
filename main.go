package main

import (
	"context"
	"crypto/tls"
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

func initZapLog() *zap.Logger {
	//config := zap.NewDevelopmentConfig()
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := config.Build()
	return logger
}

func main() {
	loggerMgr := initZapLog()
	zap.ReplaceGlobals(loggerMgr)
	defer loggerMgr.Sync() // flushes buffer, if any
	logger := loggerMgr.Sugar()

	logger.Infof("promotionChecker Version: %s, BuildDate: %s", build.Version, build.BuildDate)

	filename := getEnv("CONFIGFILE", "data.yaml")

	var item promoter.Items

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
	err = promoter.InitialRunner(&item, client, service)
	if err != nil {
		logger.DPanic(err)
		os.Exit(1)
	}

	// goroutine start the infinate runner function
	go promoter.MainRunner(ctx, &item, client, service)

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
