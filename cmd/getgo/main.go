package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	base, err := url.Parse("https://go.dev/dl/")
	if err != nil {
		log.Fatal(err)
	}

	res, err := http.Get(base.String())
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	suffix := archiveSuffix()

	doc.Find("a.download").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if val, ok := s.Attr("href"); ok {
			url, err := url.Parse(val)
			if err != nil {
				log.Fatal(err)
			}

			filename := path.Base(url.EscapedPath())

			if strings.HasSuffix(filename, suffix) {

				url := base.ResolveReference(url)

				fmt.Printf("url = %s\n", url)

				res, err := http.Get(url.String())
				if err != nil {
					log.Fatal(err)
				}

				if res.StatusCode != 200 {
					log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
				}

				zr, err := gzip.NewReader(res.Body)
				if err != nil {
					log.Fatal(err)
				}

				tr := tar.NewReader(zr)
				for {
					hdr, err := tr.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Fatal(err)
					}
					fmt.Printf("%s\n", hdr.Name)
				}

				return false
			}
		}
		return true
	})

}

func archiveSuffix() string {
	// TODO is it possible to do this without a string compare
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s-%s.zip", runtime.GOOS, runtime.GOARCH)
	} else {
		return fmt.Sprintf("%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	}
}
