package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	bmregistry "github.com/cloudfoundry/bosh-micro-cli/registry"
)

func main() {
	user := flag.String("user", "user", "User")
	password := flag.String("password", "password", "Password")
	host := flag.String("host", "127.0.0.1", "Host")
	port := flag.Int("port", 8080, "Port")
	instance := flag.String("instance", "", "Instance ID")
	settings := flag.String("settings", "", "Instance Settings")

	flag.Parse()

	logger := boshlog.NewLogger(boshlog.LevelDebug)
	serverManager := bmregistry.NewServerManager(logger)

	_, err := serverManager.Start(*user, *password, *host, *port)
	if err != nil {
		panic("Error starting registry")
	}

	if *instance != "" && *settings != "" {
		request, err := http.NewRequest(
			"PUT",
			fmt.Sprintf("http://%s:%s@%s:%d/instances/%s/settings", *user, *password, *host, *port, *instance),
			strings.NewReader(*settings),
		)

		if err != nil {
			panic("Couldn't create request")
		}

		client := http.DefaultClient
		_, err = client.Do(request)
		if err != nil {
			panic(fmt.Sprintf("Error sending request: %s", err.Error()))
		}
	}

	select {}
}
