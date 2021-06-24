package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/signintech/gopdf"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func main() {

	// create context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://www.doc88.com/p-50587088571294.html`),
		//chromedp.Click(`#continueButton`, chromedp.NodeVisible),
		//chromedp.ActionFunc(func(ctx context.Context) error {
		//	fmt.Println("Click load more to render all pages")
		//	return nil
		//}),
		//chromedp.Sleep(time.Duration(10)*time.Second),
		//chromedp.ActionFunc(func(ctx context.Context) error {
		//	// TODO: check if
		//	fmt.Println("All pages should be rendered now")
		//	return nil
		//}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < 5; i++ {
				var photo string
				chromedp.ScrollIntoView(fmt.Sprintf("#page_%d", i+1), chromedp.ByID).Do(ctx)
				chromedp.Sleep(time.Duration(1) * time.Second).Do(ctx)
				chromedp.Evaluate(fmt.Sprintf("document.querySelector('#page_%d').toDataURL()", i+1), &photo).Do(ctx)
				fmt.Println("b64: ", i+1)

				dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(photo[strings.Index(photo, ",")+1:]))
				f, err := os.Create(fmt.Sprintf("pp_%d.png", i+1))
				if err != nil {
					panic(err)
				}
				_, err = io.Copy(f, dec)
				if err != nil {
					panic(err)
				}
				f.Close()

				pdf.AddPage()
				if err := pdf.Image(fmt.Sprintf("pp_%d.png", i+1), 0, 0, gopdf.PageSizeA4); err != nil {
					panic(err)
				}
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatal("asd", err)
	}

	pdf.WritePdf("image.pdf")
}
