package snapshot

import (
	"context"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"
)

var endpoint string = "127.0.0.1:9222"

func TestSnapshot(t *testing.T) {
	s, err := NewChromeRemoteSnapshoter(endpoint)
	if err != nil {
		t.Log("screenshot:", err)
		t.Error(err.Error(), http.StatusServiceUnavailable)
	}

	var timeout time.Duration
	if timeout <= time.Second {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var url string = "https://www.google.com/"
	rd, err := s.Snapshot(ctx, url,
		ScaleFactorSnapshotOption(1),
	)
	if err != nil {
		t.Log("snapshot:", err)
		if err == context.DeadlineExceeded {
			t.Error(err.Error(), http.StatusRequestTimeout)
			return
		}
		t.Error(err.Error(), http.StatusServiceUnavailable)
		return
	}
	content, err := ioutil.ReadAll(rd)
	if err != nil {
		t.Fatal(err)
	}

	tmpfile, err := ioutil.TempFile("", "snapshot-testing")
	if err != nil {
		t.Logf("%s", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Logf("%s", err)
	}
	w := multipart.NewWriter(tmpfile)
	e := w.WriteField(tmpfile.Name()+".htm", string(content))
	t.Logf("%s", e)

	if err := tmpfile.Close(); err != nil {
		t.Logf("%s", err)
	}
}
