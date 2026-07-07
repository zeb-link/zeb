package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoJSONDecodesAPIErrorShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"COLLECTION_NOT_FOUND","message":"No collection matches."}}`))
	}))
	defer server.Close()

	client := New(Options{APIURL: server.URL, APIKey: "zeb_test"})
	err := client.DoJSON(context.Background(), http.MethodGet, "/anything", nil, &struct{}{})
	if err == nil {
		t.Fatal("expected error")
	}
	want := "COLLECTION_NOT_FOUND: No collection matches."
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestDoJSONFallsBackToRawBodyForNonAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream exploded"))
	}))
	defer server.Close()

	client := New(Options{APIURL: server.URL, APIKey: "zeb_test"})
	err := client.DoJSON(context.Background(), http.MethodGet, "/anything", nil, &struct{}{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTTP 502") || !strings.Contains(err.Error(), "upstream exploded") {
		t.Fatalf("error should carry status and body, got %q", err.Error())
	}
}

func TestDoJSONSendsAuthAndContentHeaders(t *testing.T) {
	var gotAuth, gotContentType, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := New(Options{APIURL: server.URL + "/", APIKey: "zeb_test"})
	if err := client.DoJSON(context.Background(), http.MethodPost, "/x", map[string]string{"a": "b"}, nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer zeb_test" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
}

func TestQueryStringIncludesAllSetOptions(t *testing.T) {
	got := queryString(ListLinksOptions{Limit: 100, Cursor: "abc", Sort: "total-clicks-desc", Status: "active"})
	for _, part := range []string{"limit=100", "cursor=abc", "sort=total-clicks-desc", "status=active"} {
		if !strings.Contains(got, part) {
			t.Errorf("query %q missing %q", got, part)
		}
	}
	if queryString(ListLinksOptions{}) != "" {
		t.Errorf("empty options should produce empty query, got %q", queryString(ListLinksOptions{}))
	}
}
