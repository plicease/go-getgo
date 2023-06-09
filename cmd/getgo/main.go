package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
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
			relativeUrl, err := url.Parse(val)
			if err != nil {
				log.Fatal(err)
			}

			filename := path.Base(relativeUrl.EscapedPath())

			if !strings.HasSuffix(filename, suffix) {
				return true
			}

			fmt.Printf("filename = %s\n", filename)

			installPath := installPath(filename)
			if pathExists(installPath) {
				fmt.Printf("already have this version\n")
				return false
			}

			url := base.ResolveReference(relativeUrl)

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
		return true
	})

}

func installPath(filename string) string {
	re := regexp.MustCompile(`go(?P<Version>\d+(\.\d+)+)`)

	match := re.FindStringSubmatch(filename)
	index := re.SubexpIndex("Version")

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s/opt/go/%s", home, match[index])
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	log.Fatal(err)
	return false
}

func archiveSuffix() string {
	// TODO is it possible to do this without a string compare
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s-%s.zip", runtime.GOOS, runtime.GOARCH)
	} else {
		return fmt.Sprintf("%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	}
}
