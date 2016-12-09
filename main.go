package main

import (
	"errors"
	"os"

	"github.com/golang/glog"
)

var (
	appKey    = "RymlyxWkRBKjDKsG3TpLAQ"
	secretKey = "b9dd34da8c269e44879ea1be2a0f9f7c"
	platform  = "iphone-retina"

	errMissingCredentials = errors.New("Missing email address or password")
)

func getCredentials() (string, string, error) {
	email := os.Getenv("EDGE_MAGAZINE_EMAIL")
	password := os.Getenv("EDGE_MAGAZINE_PASSWORD")

	if email != "" && password != "" {
		return email, password, nil
	}

	return "", "", errMissingCredentials
}

func main() {
	_, _, err := getCredentials()

	if err != nil {
		glog.Exit(err)
	}
}
