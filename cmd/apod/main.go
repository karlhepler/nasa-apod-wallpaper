package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

type APODImage struct {
	Name string
	URL  string
}

func main() {
	log.Println("Downloading the image and getting the filepath...")
	// Download the image and get the filepath
	filepath, err := downloadImageAndGetFilepath()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(filepath)

	log.Println("Setting the desktop background image...")
	// Set the desktop background to the image
	cmd := fmt.Sprintf("tell application \"Finder\" to set desktop picture to POSIX file \"%s\"", filepath)
	err = exec.Command("osascript", "-e", cmd).Run()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Success!")
}

func downloadImageAndGetFilepath() (string, error) {
	log.Println("Getting APOD image...")
	// Get the apod image
	img, err := getAPODImage()
	if err != nil {
		return "", err
	}

	log.Println("Getting the input URL reader...")
	// Get the input URL reader
	res, err := http.Get(img.URL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	log.Println("Creating the output file writer...")
	// Get the output file writer
	now := time.Now().Format("20060102")
	filepath := fmt.Sprintf("/Users/karl.hepler/Pictures/NASA/%s.jpg", now)
	file, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	log.Println("Copying the response body to the file...")
	// Use io.Copy to dump the response body to the file
	_, err = io.Copy(file, res.Body)
	if err != nil {
		return "", err
	}

	return filepath, nil
}

func getAPODImage() (*APODImage, error) {
	log.Println("Parsing feed URL...")
	// Parse the feed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("https://apod.nasa.gov/apod.rss")
	if err != nil {
		return nil, err
	}

	// Get the first item in the channel
	item := feed.Items[0]

	// Create the APODImage and set the name
	img := new(APODImage)
	img.Name = item.Title

	log.Println("Getting the real URL from the feed...")
	// Get the html content from the item's link
	res, err := http.Get(item.Link)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	log.Println("Getting the first img tag src url...")
	// Get the first img tag src URL
	src, err := getFirstImgTagSrcURL(res)
	if err != nil {
		return nil, err
	}

	log.Println("Resolving the full image url...")
	// The src is likely a relative URL, so resolve the full URL
	base, err := url.Parse(item.Link)
	if err != nil {
		return nil, err
	}
	img.URL = base.ResolveReference(src).String()

	return img, nil
}

func getFirstImgTagSrcURL(res *http.Response) (*url.URL, error) {
	z := html.NewTokenizer(res.Body)
	for {
		switch z.Next() {
		case html.ErrorToken:
			return nil, errors.New("Reached end of HTML body without finding a single <img /> tag!")
		case html.StartTagToken:
			t := z.Token()
			if t.Data == "img" {
				for _, a := range t.Attr {
					if a.Key == "src" {
						u, err := url.Parse(a.Val)
						if err != nil {
							return nil, err
						}
						return u, nil
					}
				}
			}
		}
	}
}
