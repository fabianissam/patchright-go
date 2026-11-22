# Patchright Go

Undetected Go version of the Playwright testing and automation library.

Patchright is a patched Playwright: this module is a drop-in replacement for
[`playwright-go`](https://github.com/playwright-community/playwright-go) that
talks to [`patchright-core`](https://www.npmjs.com/package/patchright-core)
instead of `playwright-core`.

This repository is a Go port of
[Kaliiiiiiiiii-Vinyzu/patchright-python](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright-python).
The client API is forked from playwright-go; the stealth patches match the
Python package (`patch_python_package.py`) and the patched driver.

> Patchright only patches Chromium-based browsers. Firefox and WebKit are not supported.

## Install

```bash
go get github.com/fabianissam/patchright-go
```

Install the patched driver and Chromium (or skip this and launch installed Chrome with `Channel: "chrome"`):

```bash
go run github.com/fabianissam/patchright-go/cmd/patchright@latest install chromium
```

The driver is also downloaded automatically on the first `playwright.Run()`.

## Usage

Change the import. The package name stays `playwright`, so existing
playwright-go code keeps working.

```go
package main

import (
	"log"

	playwright "github.com/fabianissam/patchright-go"
)

func main() {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatal(err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		log.Fatal(err)
	}
	page, err := browser.NewPage()
	if err != nil {
		log.Fatal(err)
	}
	if _, err = page.Goto("https://playwright.dev"); err != nil {
		log.Fatal(err)
	}
	if _, err = page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String("example.png"),
	}); err != nil {
		log.Fatal(err)
	}
	_ = browser.Close()
}
```

### Best practice — Chrome without fingerprint injection

```go
userDataDir := "./chrome-profile"
ctx, err := pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
	Channel:    playwright.String("chrome"),
	Headless:   playwright.Bool(false),
	NoViewport: playwright.Bool(true),
	// do NOT add custom browser headers or user_agent
})
```

Install Chrome with `go run github.com/fabianissam/patchright-go/cmd/patchright@latest install chrome`.

## As a Go CLI dependency

```go
import playwright "github.com/fabianissam/patchright-go"

pw, err := playwright.Run()
if err != nil {
	return err
}
defer pw.Stop()
```

`Run()` provisions the patched driver into the user cache
(`%LOCALAPPDATA%/patchright-go/<version>` on Windows). Point CLIs at an
installed Chrome via `Channel: "chrome"` if you do not want to download Chromium.

## Patches

Same client-side patches as patchright-python:

- **Runtime.enable leak** — evaluate runs in an isolated execution context by default (`isolatedContext: true`). Opt out with `playwright.EvaluateOptions{IsolatedContext: playwright.Bool(false)}`.
- **Console.enable leak** — handled in the patched driver (console APIs are disabled).
- **Command-flag leaks** — handled in the patched driver (`--enable-automation` removed, `--disable-blink-features=AutomationControlled` added, and so on).
- **Init-script injection** — `AddInitScript` / `ExposeBinding` / clock / tracing install a document route that sets `patchrightInitScript` so the driver can inject scripts without `Runtime.enable`.
- **sourceURL leak** — file-based scripts are not annotated with `//# sourceURL=...`.
- **Closed shadow roots** — handled in the patched driver.

## Extended API

```go
page.Evaluate("() => window.chrome", playwright.EvaluateOptions{
	IsolatedContext: playwright.Bool(true), // default
})

browser.NewContext(playwright.BrowserNewContextOptions{
	FocusControl: playwright.Bool(false),
})
```

See the original [Playwright documentation](https://playwright.dev/docs/intro) and the
[playwright-go API](https://pkg.go.dev/github.com/playwright-community/playwright-go).

## License

Apache-2.0, matching [patchright-python](https://github.com/Kaliiiiiiiiii-Vinyzu/patchright-python).
The Go client is derived from [playwright-go](https://github.com/playwright-community/playwright-go) (MIT); see `LICENSE.playwright-go`.
