package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gambol99/go-marathon"
	log "github.com/sirupsen/logrus"
)

var (
	// Version holds the version of this software
	Version     string
	marathonURL string
	timeout     time.Duration
	delay       time.Duration
	query       url.Values
	config      marathon.Config
	answerYes   bool
)

func init() {
	// ensure we will always have logs in logfmt format
	formatter := &log.TextFormatter{
		DisableColors: true,
	}
	log.SetFormatter(formatter)
	log.Printf("Initializing marathon-restarter %s", Version)

	var (
		waitSeconds  int
		delaySeconds int
		appIDQuery   string
		labelQuery   string
		user         string
		password     string
	)

	flag.StringVar(&marathonURL, "marathon", "http://example.com", "address of the Marathon host to query")
	flag.IntVar(&waitSeconds, "wait", 120, "timeout for a single deployment, in seconds (0 means don't wait)")
	flag.IntVar(&delaySeconds, "delay", 0, "time to sleep between restarts (0 means don't sleep)")
	flag.StringVar(&appIDQuery, "app", "", "only restart apps that contain the given string")
	flag.StringVar(&labelQuery, "label", "", "only restart apps that have the given label")
	flag.StringVar(&user, "user", "", "user for BasicAuth")
	flag.StringVar(&password, "password", "", "password for BasicAuth")
	flag.BoolVar(&answerYes, "yes", false, "assume Yes on all questions (useful for scripting)")
	flag.Parse()

	timeout = time.Duration(waitSeconds) * time.Second
	delay = time.Duration(delaySeconds) * time.Second
	query = url.Values{}
	if appIDQuery != "" {
		query.Add("id", appIDQuery)
	}
	if labelQuery != "" {
		query.Add("label", labelQuery)
	}

	config = marathon.NewDefaultConfig()
	config.URL = marathonURL
	if user != "" && password != "" {
		config.HTTPBasicAuthUser = user
		config.HTTPBasicPassword = password
		log.Printf("Using BasicAuth with user %s to authenticate.", user)
	}
}

func main() {
	log.Println("Querying Marathon for apps...")
	client, err := marathon.NewClient(config)
	if err != nil {
		log.Error(err)
		log.Error("Unable to create a Marathon client!")
		os.Exit(1)
	}

	applications, err := client.Applications(query)
	if err != nil {
		log.Error(err)
		log.Errorf("Unable to retrieve application list from Marathon at %s", marathonURL)
		os.Exit(1)
	}

	log.Printf("Found %d application(s) running", len(applications.Apps))
	if len(applications.Apps) > 0 && confirm("Do you wish to continue? [Y/n]", "y") {
		apps := applications.Apps
		for len(apps) > 0 {
			apps = restartApps(apps, client)
			if len(apps) > 0 && confirm(fmt.Sprintf("Attention! %d application(s) failed to restart, retry? [y/N]", len(apps)), "n") {
				apps = []marathon.Application{}
			}
		}
	}
}

func confirm(question string, check string) bool {
	if answerYes {
		return check == "y"
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(question)
	text, _ := reader.ReadString('\n')

	return strings.Trim(text, "\n") == check || text == "\n"
}

type marathonClient interface {
	RestartApplication(appID string, force bool) (deploymentID *marathon.DeploymentID, err error)
	WaitOnDeployment(deployID string, timeout time.Duration) (err error)
}

func restartApps(applications []marathon.Application, client marathonClient) (failedRestarts []marathon.Application) {
	for _, application := range applications {
		log.Printf("Restarting application: %v", application.ID)
		deployment, err := client.RestartApplication(application.ID, false)
		if err != nil {
			log.Error(err)
			failedRestarts = append(failedRestarts, application)
		} else if timeout > 0 {
			err = client.WaitOnDeployment(deployment.DeploymentID, timeout)
			if err != nil {
				log.Error(err)
				failedRestarts = append(failedRestarts, application)
			}
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		log.Print("Done.")
	}

	return
}
