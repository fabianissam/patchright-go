package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	playwright "github.com/fabianissam/patchright-go"
)

func main() {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start patchright: %v", err)
	}
	defer pw.Stop()

	userDataDir, err := os.MkdirTemp("", "patchright-go-profile-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(userDataDir)

	ctx, err := pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Channel:    playwright.String("chrome"),
		Headless:   playwright.Bool(true),
		NoViewport: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("could not launch chrome (is Google Chrome installed?): %v", err)
	}
	defer ctx.Close()

	pages := ctx.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = ctx.NewPage()
		if err != nil {
			log.Fatal(err)
		}
	}

	if _, err = page.Goto("https://example.com"); err != nil {
		log.Fatal(err)
	}
	title, err := page.Title()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(title)

	out := filepath.Join(".", "example.png")
	if _, err = page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(out)}); err != nil {
		log.Fatal(err)
	}
}
