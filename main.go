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
	"sort"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/net/context/ctxhttp"
)

func getCredentials() (string, string, error) {
	email := os.Getenv("EDGE_MAGAZINE_EMAIL")
	password := os.Getenv("EDGE_MAGAZINE_PASSWORD")

	if email == "" || password == "" {
		return "", "", errors.New("Missing email address or password")
	}

	return email, password, nil
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

func postForm(ctx context.Context, query string, form url.Values, result interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
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

func createAnonymousUser(ctx context.Context) (string, error) {
	form := url.Values{
		"appKey":    {"RymlyxWkRBKjDKsG3TpLAQ"},
		"secretKey": {"b9dd34da8c269e44879ea1be2a0f9f7c"},
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

func login(ctx context.Context, uid, email, password string) (string, error) {
	params := map[string]string{
		"password":   password,
		"identifier": email,
	}

	b, err := json.Marshal(params)

	if err != nil {
		return "", err
	}

	form := url.Values{
		"uid":        {uid},
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

func getDownloadURL(ctx context.Context, uid, ticket string) (int, error) {
	form := url.Values{
		"uid":    {uid},
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

func getPurchasedProductList(ctx context.Context, uid string) ([]string, error) {
	form := url.Values{
		"uid": {uid},
	}

	var response struct {
		Data struct {
			PurchasedProducts []struct {
				Product string `json:"sku"`
			} `json:"purchased_product_list"`
		} `json:"data"`
	}

	if err := postForm(ctx, "getPurchasedProductList", form, &response); err != nil {
		return nil, err
	}

	var products []string

	for _, product := range response.Data.PurchasedProducts {
		products = append(products, product.Product)
	}

	sort.Strings(products)

	return products, nil
}

func getEntitledProduct(ctx context.Context, uid, product string) (string, error) {
	form := url.Values{
		"uid": {uid},
		"sku": {product},
	}

	var response struct {
		Data struct {
			URL string `json:"secure_download_url"`
		} `json:"data"`
	}

	if err := postForm(ctx, "getEntitledProduct", form, &response); err != nil {
		return "", err
	}

	return response.Data.URL, nil
}

func main() {
	flag.Parse()

	uid, err := createAnonymousUser(context.Background())

	if err != nil {
		glog.Exit(err)
	}

	email, password, err := getCredentials()

	if err != nil {
		glog.Exit(err)
	}

	ticket, err := login(context.Background(), uid, email, password)

	if err != nil {
		glog.Exit(err)
	}

	for {
		status, err := getDownloadURL(context.Background(), uid, ticket)

		if err != nil {
			glog.Exit(err)
		}

		if status == 1 {
			break
		}
	}

	products, err := getPurchasedProductList(context.Background(), uid)

	if err != nil {
		glog.Exit(err)
	}

	for _, product := range products {
		url, err := getEntitledProduct(context.Background(), uid, product)

		if err != nil {
			glog.Exit(err)
		}

		fmt.Println(url)
	}

	glog.Flush()
}
