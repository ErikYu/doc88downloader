package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/signintech/gopdf"
	"image"
	_ "image/png"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

var doc88Link string
var A4 = &gopdf.Rect{W: 595, H: 842}

var docTitle string
var hasContinueButton bool
var pageCount int

func main() {
	flag.StringVar(&doc88Link, "link", "", "Link of doc88")
	flag.Parse()
	if doc88Link == "" {
		log.Panicln("Link cannot be empty")
	}

	if isPathExist("nDownloaderTemp") {
		if err := os.RemoveAll("nDownloaderTemp"); err != nil {
			log.Panicf("removedir failed: %v\n", err)
		}
	}
	if err := os.Mkdir("nDownloaderTemp", os.ModePerm); err != nil {
		log.Panicf("mkdir failed: %v\n", err)
	}
	fetch()
	genPdf()
}

func recursive(ctx context.Context) {
	chromedp.Click(`#continueButton`, chromedp.NodeVisible).Do(ctx)
	log.Println("Try click 'button#continueButton'.....")
	chromedp.Sleep(time.Duration(1) * time.Second)
	chromedp.Evaluate(`document.querySelectorAll('#continueButton').length === 1`, &hasContinueButton).Do(ctx)
	if hasContinueButton {
		recursive(ctx)
	} else {
		chromedp.Evaluate(`document.querySelectorAll('canvas.inner_page').length`, &pageCount).Do(ctx)
		log.Printf("%v pages found on %s", pageCount, doc88Link)
		return
	}
}

func fetch() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1080, chromedp.EmulateScale(2.0)),
		chromedp.Navigate(doc88Link),
		chromedp.Sleep(time.Duration(1)*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.Evaluate(`document.querySelectorAll('#continueButton').length === 1`, &hasContinueButton).Do(ctx)
			chromedp.Evaluate(`document.title`, &docTitle).Do(ctx)
			log.Printf("continueButton exists: %v\n", hasContinueButton)
			if hasContinueButton {
				recursive(ctx)
			} else {
				chromedp.Evaluate(`document.querySelectorAll('canvas.inner_page').length`, &pageCount).Do(ctx)
				log.Printf("%v pages found", pageCount)
			}
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < pageCount; i++ {
				var photo string
				chromedp.ScrollIntoView(fmt.Sprintf("#page_%d", i+1), chromedp.ByID).Do(ctx)
				chromedp.Sleep(time.Duration(1) * time.Second).Do(ctx)
				chromedp.Evaluate(fmt.Sprintf("document.querySelector('#page_%d').toDataURL()", i+1), &photo).Do(ctx)
				log.Println("Generating page: ", i+1)

				dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(photo[strings.Index(photo, ",")+1:]))
				f, err := os.Create(fmt.Sprintf("nDownloaderTemp/p_%d.png", i+1))
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(f, dec)
				if err != nil {
					panic(err)
				}
				f.Close()
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatal("chromedp run failed: ", err)
	}
}

func genPdf() {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *A4})

	for i := 0; i < pageCount; i++ {
		imagePath := fmt.Sprintf("nDownloaderTemp/p_%d.png", i+1)
		rect := getImageRect(imagePath)
		pageSize := &gopdf.Rect{
			W: float64(rect.Width),
			H: float64(rect.Height),
		}
		pdf.AddPageWithOption(gopdf.PageOption{
			PageSize: pageSize,
		})
		err := pdf.ImageByHolder(rect.Holder, 0, 0, pageSize)
		if err != nil {
			log.Panic(err)
		}
	}
	if err := pdf.WritePdf(fmt.Sprintf("%s.pdf", docTitle)); err != nil {
		log.Panicf("Generate pdf failed: %v", err)
	}
	log.Println("Removing cache dir")
	_ = os.RemoveAll("nDownloaderTemp")
	log.Println("Pdf generated successfully")
}

type ImageForPDF struct {
	Width, Height int
	Holder        gopdf.ImageHolder
}

func isPathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			panic(err)
		}
	} else {
		return true
	}
}

func getImageRect(imagePath string) *ImageForPDF {
	file, err := os.Open(imagePath)
	defer file.Close()
	if err != nil {
		log.Panicf("Open %s failed: %v", imagePath, err)
	}
	im, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Panicf("Fetch config %s failed: %v", imagePath, err)
	}
	holder, err := gopdf.ImageHolderByPath(imagePath)
	if err != nil {
		log.Panicf("Generate imageHolder %s failed: %v", imagePath, err)
	}
	return &ImageForPDF{
		Width:  im.Width,
		Height: im.Height,
		Holder: holder,
	}
}
