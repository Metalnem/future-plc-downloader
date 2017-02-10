package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/net/context/ctxhttp"
)

func dumpResponse(resp *http.Response) {
	if glog.V(1) {
		body, err := httputil.DumpResponse(resp, true)

		if err != nil {
			glog.Fatal(err)
		}

		glog.V(1).Infof("%s", body)
	}
}

func postForm(ctx context.Context, query string, form url.Values, result interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.futr.efs.foliocloud.net/%s/", query)
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

	var errs struct {
		Errors interface{} `json:"errors"`
	}

	if err = json.Unmarshal(b, &errs); err != nil {
		return err
	}

	if errs, ok := errs.Errors.(map[string]interface{}); ok {
		for key, value := range errs {
			return errors.Errorf("%s: %v", key, value)
		}
	}

	return json.Unmarshal(b, result)
}

// Session contains session data for single user and single magazine.
type Session struct {
	uid string
	mag magazine
}

func createAnonymousUser(ctx context.Context, mag magazine) (string, error) {
	form := url.Values{
		"appKey":    {mag.appKey},
		"secretKey": {mag.secretKey},
		"platform":  {"iphone-retina"},
	}

	var response struct {
		Data struct {
			UID string `json:"uid"`
		} `json:"data"`
	}

	if err := postForm(ctx, "createAnonymousUser", form, &response); err != nil {
		return "", err
	}

	return response.Data.UID, nil
}

func getProductList(ctx context.Context, uid string) ([]string, error) {
	form := url.Values{
		"uid": {uid},
	}

	var response struct {
		Data struct {
			Products []struct {
				ID string `json:"sku"`
			} `json:"product_list"`
		} `json:"data"`
	}

	if err := postForm(ctx, "getProductList", form, &response); err != nil {
		return nil, err
	}

	var ids []string

	for _, product := range response.Data.Products {
		ids = append(ids, product.ID)
	}

	return ids, nil
}

// NewSession creates new unauthenticated session on the server.
func NewSession(ctx context.Context, mag magazine) (*Session, error) {
	for {
		uid, err := createAnonymousUser(ctx, mag)

		if err != nil {
			return nil, err
		}

		// Response from the createAnonymousUser call can contain
		// invalid UID. We have to call some other endpoint that is
		// using the UID to check if it can be authenticated.

		_, err = getProductList(ctx, uid)

		if err != nil && err.Error() == "AUT002: could not authenticate uid" {
			continue
		}

		if err != nil {
			return nil, err
		}

		return &Session{uid: uid, mag: mag}, nil
	}
}

// RestoreSession restores session that was initialized on some other device.
func RestoreSession(ctx context.Context, mag magazine, uid string) (*Session, error) {
	_, err := getProductList(ctx, uid)

	if err != nil {
		return nil, err
	}

	return &Session{uid: uid, mag: mag}, nil
}

func (s *Session) login(ctx context.Context, email, password string) (string, error) {
	params := map[string]string{
		"password":   password,
		"identifier": email,
	}

	b, err := json.Marshal(params)

	if err != nil {
		return "", err
	}

	form := url.Values{
		"uid":        {s.uid},
		"api_params": {string(b)},
	}

	var response struct {
		Data struct {
			Ticket string `json:"download_ticket_no"`
		} `json:"data"`
	}

	if err = postForm(ctx, "login", form, &response); err != nil {
		return "", err
	}

	return response.Data.Ticket, nil
}

func (s *Session) getDownloadURL(ctx context.Context, ticket string) (int, error) {
	form := url.Values{
		"uid":    {s.uid},
		"ticket": {ticket},
	}

	var response struct {
		Data struct {
			Status int `json:"status"`
		} `json:"data"`
	}

	if err := postForm(ctx, "getDownloadUrl", form, &response); err != nil {
		return 0, err
	}

	return response.Data.Status, nil
}

// Login authenticates existing session with the given email and password.
func (s *Session) Login(ctx context.Context, email, password string) error {
	ticket, err := s.login(ctx, email, password)

	if err != nil {
		return err
	}

	for {
		status, err := s.getDownloadURL(ctx, ticket)

		if err != nil {
			return err
		}

		if status == 1 {
			return nil
		}
	}
}
