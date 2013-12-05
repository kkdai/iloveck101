package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/coreos/go-log/log"
)

var (
	dir      string
	threadId = regexp.MustCompile(`thread-(\d*)-`)
	imageId  = regexp.MustCompile(`([^\/]+)\.(png|jpg)`)
)

func worker(linkChan chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for url := range linkChan {
		resp, err := http.Get(url)
		if err != nil {
			log.Println(err)
			continue
		}
		defer resp.Body.Close()

		m, _, err := image.Decode(resp.Body)
		if err != nil {
			log.Println(err)
			continue
		}

		// Ignore small images
		bounds := m.Bounds()
		if bounds.Size().X > 300 && bounds.Size().Y > 300 {
			imgInfo := imageId.FindStringSubmatch(url)
			out, _ := os.Create(dir + "/" + imgInfo[1] + "." + imgInfo[2])
			defer out.Close()
			switch imgInfo[2] {
			case "jpg":
				jpeg.Encode(out, m, nil)
			case "png":
				png.Encode(out, m)
			}
		}
	}
}

func main() {
	var url string
	var workers int
	flag.StringVar(&url, "u", "http://ck101.com/thread-2876990-1-1.html", "Destination")
	flag.IntVar(&workers, "w", 10, "Workers number")
	flag.Parse()

	doc, err := goquery.NewDocument(url)
	if err != nil {
		panic(err)
	}

	usr, _ := user.Current()
	title := doc.Find("h1#thread_subject").Text()
	dir = fmt.Sprintf("%v/Pictures/iloveck101/%v - %v", usr.HomeDir, threadId.FindStringSubmatch(url)[1], title)

	os.MkdirAll(dir, 0755)

	linkChan := make(chan string)
	wg := new(sync.WaitGroup)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(linkChan, wg)
	}

	doc.Find("div[itemprop=articleBody] img").Each(func(i int, img *goquery.Selection) {
		imgUrl, _ := img.Attr("file")
		linkChan <- imgUrl
	})

	close(linkChan)
	wg.Wait()
}
