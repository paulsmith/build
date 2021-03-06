// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gerrit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// taken from https://go-review.googlesource.com/projects/go
var exampleProjectResponse = []byte(`)]}'
{
  "id": "go",
  "name": "go",
  "parent": "All-Projects",
  "description": "The Go Programming Language",
  "state": "ACTIVE",
  "web_links": [
    {
      "name": "gitiles",
      "url": "https://go.googlesource.com/go/",
      "target": "_blank"
    }
  ]
}
`)

func TestGetProjectInfo(t *testing.T) {
	hitServer := false
	path := ""
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitServer = true
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(200)
		w.Write(exampleProjectResponse)
	}))
	defer s.Close()
	c := NewClient(s.URL, NoAuth)
	info, err := c.GetProjectInfo(context.Background(), "go")
	if err != nil {
		t.Fatal(err)
	}
	if !hitServer {
		t.Errorf("expected to hit test server, didn't")
	}
	if path != "/projects/go" {
		t.Errorf("expected Path to be '/projects/go', got %s", path)
	}
	if info.Name != "go" {
		t.Errorf("expected Name to be 'go', got %s", info.Name)
	}
}

func TestProjectNotFound(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(404)
		w.Write([]byte("Not found: unknown"))
	}))
	defer s.Close()
	c := NewClient(s.URL, NoAuth)
	_, err := c.GetProjectInfo(context.Background(), "unknown")
	if err != ErrProjectNotExist {
		t.Errorf("expected to get ErrProjectNotExist, got %v", err)
	}
}

func TestContextError(t *testing.T) {
	c := NewClient("http://localhost", NoAuth)
	yearsAgo, _ := time.Parse("2006", "2006")
	ctx, cancel := context.WithDeadline(context.Background(), yearsAgo)
	defer cancel()
	_, err := c.GetProjectInfo(ctx, "unknown")
	if err == nil {
		t.Errorf("expected non-nil error, got nil")
	}
	uerr, ok := err.(*url.Error)
	if !ok {
		t.Errorf("expected url.Error, got %#v", err)
	}
	if uerr.Err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", uerr.Err)
	}
}

var getChangeResponse = []byte(`)]}'
{
  "id": "build~master~I92989e0231299ed305ddfbbe6034d293f1c87470",
  "project": "build",
  "branch": "master",
  "hashtags": [],
  "change_id": "I92989e0231299ed305ddfbbe6034d293f1c87470",
  "subject": "devapp: fix tests",
  "status": "ABANDONED",
  "created": "2017-07-13 06:09:10.000000000",
  "updated": "2017-07-14 16:31:32.000000000",
  "insertions": 1,
  "deletions": 1,
  "unresolved_comment_count": 0,
  "has_review_started": true,
  "_number": 48330,
  "owner": {
    "_account_id": 13437
  },
  "messages": [
    {
      "id": "f9fcf0ff9eb58fc8edd989f8bbb3500ff73f9b11",
      "author": {
        "_account_id": 22285
      },
      "real_author": {
        "_account_id": 22285
      },
      "date": "2017-07-13 06:14:48.000000000",
      "message": "Patch Set 1:\n\nCheck out https://go-review.googlesource.com/c/48350/ :)",
      "_revision_number": 1
    }
  ]
}`)

func TestGetChange(t *testing.T) {
	hitServer := false
	uri := ""
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitServer = true
		uri = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(200)
		w.Write(getChangeResponse)
	}))
	defer s.Close()
	c := NewClient(s.URL, NoAuth)
	info, err := c.GetChange(context.Background(), "48330", QueryChangesOpt{
		Fields: []string{"MESSAGES"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hitServer {
		t.Errorf("expected to hit test server, didn't")
	}
	if want := "/changes/48330?o=MESSAGES"; uri != want {
		t.Errorf("expected RequestURI to be %q, got %q", want, uri)
	}
	if len(info.Messages) != 1 {
		t.Errorf("expected message length to be 1, got %d", len(info.Messages))
	}
	msg := info.Messages[0].Message
	if !strings.Contains(msg, "Check out") {
		t.Errorf("expected to find string in Message, got %s", msg)
	}
}

func TestGetChangeError(t *testing.T) {
	hitServer := false
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitServer = true
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.WriteHeader(404)
		io.WriteString(w, "Not found: 99999")
	}))
	defer s.Close()
	c := NewClient(s.URL, NoAuth)
	_, err := c.GetChange(context.Background(), "99999", QueryChangesOpt{
		Fields: []string{"MESSAGES"},
	})
	if err != ErrChangeNotExist {
		t.Errorf("expected ErrChangeNotExist, got %v", err)
	}
}
