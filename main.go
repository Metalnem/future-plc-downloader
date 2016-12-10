package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/net/context/ctxhttp"
)

var (
	baseURL     = "https://api.futr.efs.foliocloud.net"
	httpTimeout = 30 * time.Second

	errMissingCredentials = errors.New("Missing email address or password")
)

type createAnonymousUserRespone struct {
	Data struct {
		UID    string   `json:"uid"`
		Errors []string `json:"errors"`
	} `json:"data"`
}

func getCredentials() (string, string, error) {
	email := os.Getenv("EDGE_MAGAZINE_EMAIL")
	password := os.Getenv("EDGE_MAGAZINE_PASSWORD")

	if email != "" && password != "" {
		return email, password, nil
	}

	return "", "", errMissingCredentials
}

func dumpResponse(resp *http.Response) {
	if glog.V(1) {
		body, err := httputil.DumpResponse(resp, true)

		if err != nil {
			glog.Fatal(err)
		}

		glog.V(1).Infof("%s", body)
	}
}

func postForm(ctx context.Context, url string, form url.Values, data interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	resp, err := ctxhttp.PostForm(ctx, http.DefaultClient, url, form)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	dumpResponse(resp)

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	return json.Unmarshal(b, data)
}

func createAnonymousUser(ctx context.Context) (string, error) {
	form := url.Values{
		"appKey":    {"RymlyxWkRBKjDKsG3TpLAQ"},
		"secretKey": {"b9dd34da8c269e44879ea1be2a0f9f7c"},
		"platform":  {"iphone-retina"},
	}

	var data createAnonymousUserRespone

	if err := postForm(ctx, baseURL+"/createAnonymousUser/", form, &data); err != nil {
		return "", err
	}

	return data.Data.UID, nil
}

func main() {
	flag.Parse()

	uid, err := createAnonymousUser(context.Background())

	if err != nil {
		glog.Exit(err)
	}

	fmt.Println(uid)
	glog.Flush()
}
