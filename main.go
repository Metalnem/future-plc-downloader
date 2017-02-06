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

const usage = `Usage of Edge Magazine downloader:
  -list
    	List all available issues
  -all
    	Download all available issues
  -from
    	Download all issues starting with the specified ID
  -single int
    	Download single issue with the specified ID
  -email string
    	Account email
  -password string
    	Account password
  -uid string
    	Account UID`

var (
	pdfPassword = []byte(`"F0rd*t3h%3p1c&h0nkY!"`)

	errDecryptionFailed   = errors.New("Failed to decrypt PDF page")
	errInvalidIssueNumber = errors.New("Invalid issue number received from server")
	errInvalidPageNumber  = errors.New("Invalid page number received from server")
	errIssueDoesNotExist  = errors.New("Specified issue does not exist in your library")
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

func getCredentials(email, password string) (string, string, error) {
	if email != "" && password != "" {
		return email, password, nil
	}

	email = os.Getenv("EDGE_MAGAZINE_EMAIL")
	password = os.Getenv("EDGE_MAGAZINE_PASSWORD")

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

func (m magazine) createAnonymousUser(ctx context.Context) (string, error) {
	form := url.Values{
		"appKey":    {m.appKey},
		"secretKey": {m.secretKey},
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

func (m magazine) getIssueNumber(name string) (int, error) {
	var n int

	if _, err := fmt.Sscanf(name, fmt.Sprintf("com.futurenet.%s.%s", m.id, "%d"), &n); err != nil {
		return 0, err
	}

	if n <= 0 {
		return 0, errInvalidIssueNumber
	}

	return n, nil
}

func (m magazine) getIssue(ctx context.Context, uid, product string) (issue, error) {
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

	n, err := m.getIssueNumber(response.Data.ID)

	if err != nil {
		return issue{}, err
	}

	return issue{Title: response.Data.Title, Number: n, URL: response.Data.URL}, nil
}

func (m magazine) getIssues(ctx context.Context, email, password, uid string) ([]issue, error) {
	var err error

	if uid == "" {
		uid, err = m.createAnonymousUser(ctx)
	}

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
		issue, err := m.getIssue(ctx, uid, id)

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

func save(issue issue, path string) (err error) {
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

	f, err := os.Create(path)

	if err != nil {
		return err
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return w.Write(f)
}

func listAll(issues []issue) error {
	for _, issue := range issues {
		if _, err := fmt.Printf("%d. %s\n", issue.Number, issue.Title); err != nil {
			return err
		}
	}

	return nil
}

func (m magazine) downloadAll(issues []issue) error {
	_, err := m.downloadFunc(issues, func(issue) bool {
		return true
	})

	return err
}

func (m magazine) downloadFrom(issues []issue, number int) error {
	_, err := m.downloadFunc(issues, func(issue issue) bool {
		return issue.Number >= number
	})

	return err
}

func (m magazine) downloadSingle(issues []issue, number int) error {
	count, err := m.downloadFunc(issues, func(issue issue) bool {
		return issue.Number == number
	})

	if err != nil {
		return err
	}

	if count == 0 {
		return errIssueDoesNotExist
	}

	return nil
}

func (m magazine) downloadFunc(issues []issue, f func(issue) bool) (int, error) {
	count := 0

	for _, issue := range issues {
		if f(issue) {
			path := fmt.Sprintf("%s %d (%s).pdf", m.name, issue.Number, issue.Title)
			temp := fmt.Sprintf("%s.part", path)

			if err := save(issue, temp); err != nil {
				return 0, err
			}

			if err := os.Rename(temp, path); err != nil {
				return 0, err
			}

			count++
		}
	}

	return count, nil
}

func main() {
	list := flag.Bool("list", false, "List all available issues")
	all := flag.Bool("all", false, "Download all available issues")
	from := flag.Int("from", 0, "Download all issues starting with the specified ID")
	single := flag.Int("single", 0, "Download single issue with the specified ID")

	var email, password, uid string

	flag.StringVar(&email, "email", "", "Account email")
	flag.StringVar(&password, "password", "", "Account password")
	flag.StringVar(&uid, "uid", "", "Account UID")

	flag.Parse()

	if !*list && !*all && *from <= 0 && *single <= 0 {
		fmt.Println(usage)
		os.Exit(1)
	}

	email, password, err := getCredentials(email, password)

	if err != nil {
		fmt.Println(usage)
		os.Exit(1)
	}

	m := magazine{"Edge", "RymlyxWkRBKjDKsG3TpLAQ", "b9dd34da8c269e44879ea1be2a0f9f7c", "edgemagazine"}
	issues, err := m.getIssues(context.Background(), email, password, uid)

	if err != nil {
		glog.Exit(err)
	}

	switch {
	case *list:
		err = listAll(issues)
	case *all:
		err = m.downloadAll(issues)
	case *from > 0:
		err = m.downloadFrom(issues, *from)
	case *single > 0:
		err = m.downloadSingle(issues, *single)
	}

	if err != nil {
		glog.Exit(err)
	}

	glog.Flush()
}
