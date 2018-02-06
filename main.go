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
	marathonURL string
	limit       int
	timeout     time.Duration
	query       url.Values
)

func init() {
	var wait int
	var appIDQuery string
	var labelQuery string
	flag.StringVar(&marathonURL, "marathon", "http://example.com", "address of the Marathon host to query")
	flag.IntVar(&limit, "limit", 1, "maximum number of apps restarted at once")
	flag.IntVar(&wait, "wait", 120, "timeout for a single deployment, in seconds")
	flag.StringVar(&appIDQuery, "app", "", "only restart apps that contain the given string")
	flag.StringVar(&labelQuery, "label", "", "only restart apps that have the given label")
	flag.Parse()

	timeout = time.Duration(wait) * time.Second
	query = url.Values{}
	if appIDQuery != "" {
		query.Add("id", appIDQuery)
	}
	if labelQuery != "" {
		query.Add("label", labelQuery)
	}

	// ensure we will always have logs in logfmt format
	formatter := &log.TextFormatter{
		DisableColors: true,
	}
	log.SetFormatter(formatter)
}

func main() {
	log.Println("Querying Marathon for apps...")
	config := marathon.NewDefaultConfig()
	config.URL = marathonURL
	client, err := marathon.NewClient(config)
	if err != nil {
		log.Error(err)
		log.Panic("Unable to create a Marathon client!")
	}

	applications, err := client.Applications(query)
	if err != nil {
		log.Error(err)
		log.Panicf("Unable to retrieve application list from Marathon at %s", marathonURL)
	}

	log.Printf("Found %d applications running", len(applications.Apps))
	if confirm("Do you wish to continue? [Y/n]", "y") {
		apps := applications.Apps
		for len(apps) > 0 {
			apps = restartApps(apps, client)
			if len(apps) > 0 && confirm(fmt.Sprintf("Attention! %d applications failed to restart, retry? [y/N]", len(apps)), "n") {
				apps = []marathon.Application{}
			}
		}
	}
}

func confirm(question string, check string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(question)
	text, _ := reader.ReadString('\n')

	return strings.Trim(text, "\n") == check || text == "\n"
}

func restartApps(applications []marathon.Application, client marathon.Marathon) (failedRestarts []marathon.Application) {
	for _, application := range applications {
		log.Printf("Restarting application: %v", application.ID)
		deployment, err := client.RestartApplication(application.ID, false)
		if err != nil {
			log.Error(err)
			failedRestarts = append(failedRestarts, application)
		} else {
			err = client.WaitOnDeployment(deployment.DeploymentID, timeout)
			if err != nil {
				log.Error(err)
				failedRestarts = append(failedRestarts, application)
			}
		}
		log.Print("Done.")
	}

	return
}
