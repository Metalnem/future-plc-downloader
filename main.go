package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/unidoc/unidoc/pdf"
	"golang.org/x/net/context/ctxhttp"
)

var (
	pdfPassword = []byte(`"F0rd*t3h%3p1c&h0nkY!"`)

	errDecryptionFailed   = errors.New("Failed to decrypt PDF page")
	errInvalidIssueNumber = errors.New("Invalid issue number received from server")
	errInvalidPageNumber  = errors.New("Invalid page number received from server")
	errNoPages            = errors.New("No pages found in archive")
	errMissingCredentials = errors.New("Missing email address or password")
)

type issue struct {
	Title  string
	Number int
	URL    string
}

type page struct {
	*bytes.Reader
}

func getCredentials() (string, string, error) {
	email := os.Getenv("EDGE_MAGAZINE_EMAIL")
	password := os.Getenv("EDGE_MAGAZINE_PASSWORD")

	if email == "" || password == "" {
		return "", "", errMissingCredentials
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
				ID string `json:"sku"`
			} `json:"purchased_product_list"`
		} `json:"data"`
	}

	if err := postForm(ctx, "getPurchasedProductList", form, &response); err != nil {
		return nil, err
	}

	var ids []string

	for _, product := range response.Data.PurchasedProducts {
		ids = append(ids, product.ID)
	}

	sort.Strings(ids)

	return ids, nil
}

func getIssueNumber(name string) (int, error) {
	var n int

	if _, err := fmt.Sscanf(name, "com.futurenet.edgemagazine.%d", &n); err != nil {
		return 0, err
	}

	if n <= 0 {
		return 0, errInvalidIssueNumber
	}

	return n, nil
}

func getIssue(ctx context.Context, uid, product string) (issue, error) {
	form := url.Values{
		"uid": {uid},
		"sku": {product},
	}

	var response struct {
		Data struct {
			ID    string `json:"sku"`
			Title string `json:"product_title"`
			URL   string `json:"secure_download_url"`
		} `json:"data"`
	}

	if err := postForm(ctx, "getEntitledProduct", form, &response); err != nil {
		return issue{}, err
	}

	n, err := getIssueNumber(response.Data.ID)

	if err != nil {
		return issue{}, err
	}

	return issue{Title: response.Data.Title, Number: n, URL: response.Data.URL}, nil
}

func getIssues(ctx context.Context, email, password string) ([]issue, error) {
	uid, err := createAnonymousUser(ctx)

	if err != nil {
		return nil, err
	}

	ticket, err := login(ctx, uid, email, password)

	if err != nil {
		return nil, err
	}

	newCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		status, err := getDownloadURL(newCtx, uid, ticket)

		if err != nil {
			return nil, err
		}

		if status == 1 {
			break
		}
	}

	ids, err := getPurchasedProductList(ctx, uid)

	if err != nil {
		return nil, err
	}

	var issues []issue

	for _, id := range ids {
		issue, err := getIssue(ctx, uid, id)

		if err != nil {
			return nil, err
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

func download(ctx context.Context, url string) (*zip.Reader, error) {
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(b)
	l := int64(r.Len())

	return zip.NewReader(r, l)
}

func getPageNumber(name string) (int, error) {
	var n int

	if _, err := fmt.Sscanf(name, "page-%d.pdf", &n); err != nil {
		return 0, err
	}

	if n < 0 {
		return 0, errInvalidPageNumber
	}

	return n, nil
}

func getPages(zr *zip.Reader) ([]page, error) {
	pages := make(map[int]page)

	for _, f := range zr.File {
		if filepath.Ext(f.Name) != ".pdf" {
			continue
		}

		n, err := getPageNumber(f.Name)

		if err != nil {
			return nil, err
		}

		fr, err := f.Open()

		if err != nil {
			return nil, err
		}

		defer fr.Close()

		b, err := ioutil.ReadAll(fr)

		if err != nil {
			return nil, err
		}

		pages[n] = page{bytes.NewReader(b)}
	}

	if len(pages) == 0 {
		return nil, errNoPages
	}

	var result []page

	for i := 0; i < len(pages); i++ {
		page, ok := pages[i]

		if !ok {
			return nil, errors.Errorf("Page %d is missing", i)
		}

		result = append(result, page)
	}

	return result, nil
}

func unlockAndMerge(pages []page) (*pdf.PdfWriter, error) {
	w := pdf.NewPdfWriter()

	for _, page := range pages {
		r, err := pdf.NewPdfReader(page)

		if err != nil {
			return nil, err
		}

		ok, err := r.Decrypt(pdfPassword)

		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, errDecryptionFailed
		}

		numPages, err := r.GetNumPages()

		if err != nil {
			return nil, err
		}

		for i := 0; i < numPages; i++ {
			page, err := r.GetPage(i + 1)

			if err != nil {
				return nil, err
			}

			if err = w.AddPage(page); err != nil {
				return nil, err
			}
		}
	}

	return &w, nil
}

func save(issue issue) error {
	zr, err := download(context.Background(), issue.URL)

	if err != nil {
		return err
	}

	pages, err := getPages(zr)

	if err != nil {
		return err
	}

	w, err := unlockAndMerge(pages)

	if err != nil {
		return err
	}

	path := fmt.Sprintf("Edge Magazine %d (%s).pdf", issue.Number, issue.Title)
	f, err := os.Create(path)

	if err != nil {
		return err
	}

	defer f.Close()

	return w.Write(f)
}

func main() {
	flag.Parse()

	email, password, err := getCredentials()

	if err != nil {
		glog.Exit(err)
	}

	issues, err := getIssues(context.Background(), email, password)

	if err != nil {
		glog.Exit(err)
	}

	for _, issue := range issues {
		if err = save(issue); err != nil {
			glog.Exit(err)
		}
	}

	glog.Flush()
}
