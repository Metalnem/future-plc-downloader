package main

import (
	"archive/zip"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/unidoc/unidoc/pdf"
)

const usage = `Usage of Future plc downloader:
  -name
    	Magazine name
  -email string
    	Account email
  -password string
    	Account password
  -uid string
    	Account UID`

var (
	pdfPassword = []byte(`"F0rd*t3h%3p1c&h0nkY!"`)

	errUnknownMagazine   = errors.New("Unknown magazine name")
	errDecryptionFailed  = errors.New("Failed to decrypt PDF page")
	errInvalidPageNumber = errors.New("Invalid page number received from server")
	errIssueDoesNotExist = errors.New("Specified issue does not exist in your library")
	errNoPages           = errors.New("No pages found in archive")
)

type issue struct {
	Title string
	URL   string
}

type page struct {
	*bytes.Reader
}

func getValue(value, env string) string {
	if value != "" {
		return value
	}

	return os.Getenv(env)
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

	return ids, nil
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

	return issue{Title: response.Data.Title, URL: response.Data.URL}, nil
}

func (s *Session) getIssues(ctx context.Context) ([]issue, error) {
	ids, err := getPurchasedProductList(ctx, s.uid)

	if err != nil {
		return nil, err
	}

	sort.Strings(ids)
	var issues []issue

	for _, id := range ids {
		issue, err := s.mag.getIssue(ctx, s.uid, id)

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
		ext := filepath.Ext(f.Name)
		base := filepath.Base(f.Name)

		if ext != ".pdf" || base[0] == '.' {
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

		encrypted, err := r.IsEncrypted()

		if err != nil {
			return nil, err
		}

		if encrypted {
			ok, err := r.Decrypt(pdfPassword)

			if err != nil {
				return nil, err
			}

			if !ok {
				return nil, errDecryptionFailed
			}
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

func save(ctx context.Context, issue issue, path string) (err error) {
	zr, err := download(ctx, issue.URL)

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

func printUsage() {
	fmt.Println(usage)
	fmt.Println()
	fmt.Println("Supported magazines:")

	for _, mag := range magazines {
		fmt.Printf("  %s\n", mag.name)
	}
}

func downloadAll(ctx context.Context, m magazine, issues []issue) {
	for _, issue := range issues {
		path := fmt.Sprintf("%s %s.pdf", m.name, issue.Title)
		temp := fmt.Sprintf("%s.part", path)

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			continue
		}

		if err := save(ctx, issue, temp); err != nil {
			glog.Error(err)
			continue
		}

		if err := os.Rename(temp, path); err != nil {
			glog.Error(err)
			continue
		}
	}
}

func main() {
	var name, email, password, uid string

	flag.StringVar(&name, "name", "", "Magazine name")
	flag.StringVar(&email, "email", "", "Account email")
	flag.StringVar(&password, "password", "", "Account password")
	flag.StringVar(&uid, "uid", "", "Account UID")

	flag.Parse()

	email = getValue(email, "EDGE_MAGAZINE_EMAIL")
	password = getValue(password, "EDGE_MAGAZINE_PASSWORD")
	uid = getValue(uid, "EDGE_MAGAZINE_UID")

	if name == "" || email == "" || password == "" {
		printUsage()
		os.Exit(1)
	}

	mag, ok := getMagazine(name)

	if !ok {
		glog.Exit(errUnknownMagazine)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var s *Session
	var err error

	if uid == "" {
		s, err = NewSession(ctx, mag)
	} else {
		s, err = RestoreSession(ctx, mag, uid)
	}

	if err != nil {
		glog.Exit(err)
	}

	if err = s.Login(ctx, email, password); err != nil {
		glog.Exit(err)
	}

	issues, err := s.getIssues(ctx)

	if err != nil {
		glog.Exit(err)
	}

	downloadAll(context.Background(), mag, issues)
	glog.Flush()
}
