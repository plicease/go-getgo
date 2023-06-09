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
	"path/filepath"
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

				path := localInstallPath(installPath, hdr.Name)
				fmt.Println(path)

				switch hdr.Typeflag {
				case tar.TypeDir:
					if err := os.MkdirAll(path, 0755); err != nil {
						log.Fatalf("error creating directory %s: %v", path, err)
					}
				case tar.TypeReg:
					dir := filepath.Dir(path)
					if err := os.MkdirAll(dir, 0755); err != nil {
						log.Fatalf("error creating directory %s: %s", dir, err)
					}
					file, err := os.Create(path)
					file.Chmod(os.FileMode(hdr.Mode))
					if err != nil {
						log.Fatalf("error opening %s: %s", path, err)
					}
					if _, err := io.Copy(file, tr); err != nil {
						log.Fatalf("error writing %s: %s", path, err)
					}
					file.Close()
				default:
					log.Fatalf("unknown type for %s", path)
				}
			}

			symlinkName := filepath.Clean(installPath + "/../.path")

			if pathExists(symlinkName) {
				err := os.Remove(symlinkName)
				if err != nil {
					log.Fatalf("unable to remove %s: %s", symlinkName, err)
				}
			}

			symlinkTarget := filepath.Base(installPath)

			if err := os.Symlink(symlinkTarget, symlinkName); err != nil {
				log.Fatalf("unable to create symlink %s %s: %s", symlinkTarget, symlinkName, err)
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

func localInstallPath(installPath string, archivePath string) string {
	s := (strings.SplitN(archivePath, "/", 2))[1]
	return installPath + "/" + s
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
