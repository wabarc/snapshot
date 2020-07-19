package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Snapshoter is a webpage snapshot interface.
type Snapshoter interface {
	Snapshot(ctx context.Context, url string, options ...SnapshotOption) (io.Reader, error)
}

type chromeRemoteSnapshoter struct {
	url string
}

// NewChromeRemoteSnapshoter creates a Snapshoter backed by Chrome DevTools Protocol.
// The addr is the headless chrome websocket debugger endpoint, such as 127.0.0.1:9222.
func NewChromeRemoteSnapshoter(addr string) (Snapshoter, error) {
	// Due to issue#505 (https://github.com/chromedp/chromedp/issues/505),
	// chrome restricts the host must be IP or localhost, we should rewrite the url.
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/json/version", addr), nil)
	if err != nil {
		return nil, err
	}
	req.Host = "localhost"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &chromeRemoteSnapshoter{
		url: strings.Replace(result["webSocketDebuggerUrl"].(string), "localhost", addr, 1),
	}, nil
}

func (s *chromeRemoteSnapshoter) Snapshot(ctx context.Context, url string, options ...SnapshotOption) (io.Reader, error) {
	allocatorCtx, cancel := chromedp.NewRemoteAllocator(ctx, s.url)
	defer cancel()

	ctxt, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	var opts SnapshotOptions
	for _, o := range options {
		o(&opts)
	}

	var buf []byte
	captureAction := s.snapshotAction(&buf, opts.Format)
	if opts.Format == "pdf" {
		captureAction = s.printToPDFAction(&buf, page.PrintToPDF())
	}

	err := chromedp.Run(ctxt,
		emulation.SetDeviceMetricsOverride(opts.Width, opts.Height, opts.ScaleFactor, opts.Mobile),
		chromedp.Navigate(url),
		captureAction,
		s.closePageAction(),
	)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(buf), nil
}

func (s *chromeRemoteSnapshoter) printToPDFAction(res *[]byte, params *page.PrintToPDFParams) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		if res == nil {
			return
		}

		if params == nil {
			params = page.PrintToPDF()
		}

		*res, _, err = params.Do(ctx)
		return
	})
}

func (s *chromeRemoteSnapshoter) snapshotAction(res *[]byte, format string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		if res == nil {
			return
		}

		params := page.CaptureSnapshot()
		// switch format {
		// case "pdf":
		// 	// TODO
		// 	// params.Format = page.CaptureSnapshotFormatJpeg
		// case "mhtml", "mhtm":
		// default:
		// }
		if data, err := params.Do(ctx); err == nil {
			*res = []byte(data)
		}
		return
	})
}

func (s *chromeRemoteSnapshoter) closePageAction() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) (err error) {
		return page.Close().Do(ctx)
	})
}

// SnapshotOptions is the options used by Snapshot.
type SnapshotOptions struct {
	Width  int64
	Height int64
	Mobile bool
	Format string // mhtml, pdf, default mhtml.

	ScaleFactor float64
}

type SnapshotOption func(*SnapshotOptions)

func WidthSnapshotOption(width int64) SnapshotOption {
	return func(opts *SnapshotOptions) {
		opts.Width = width
	}
}

func HeightSnapshotOption(height int64) SnapshotOption {
	return func(opts *SnapshotOptions) {
		opts.Height = height
	}
}

func ScaleFactorSnapshotOption(factor float64) SnapshotOption {
	return func(opts *SnapshotOptions) {
		opts.ScaleFactor = factor
	}
}

func MobileSnapshotOption(b bool) SnapshotOption {
	return func(opts *SnapshotOptions) {
		opts.Mobile = b
	}
}

func FormatSnapshotOption(format string) SnapshotOption {
	return func(opts *SnapshotOptions) {
		opts.Format = format
	}
}
