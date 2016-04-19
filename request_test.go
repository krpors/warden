package main

import (
	"strings"
	"testing"
)

// Tests a normal request, with a normal front-matter plus some body content.
func TestNewRequestNormal(t *testing.T) {
	doc := `{
    "name": "Example org-testing---",
    "url": "http://example.org/some-uri-with-dashes",
    "method": "GET",
	"timeout": 10000,
    "headers": [
        "Authorization: Basic bG9sbGVyY29hc3Rlcgo=",
        "Some-HTTP-Header: Blah blah"
    ]
}
---
This is the data to send to the URL given above.`

	reader := strings.NewReader(doc)
	req, err := newRequest(reader)
	if err != nil {
		t.Errorf("failed parsing request: %v", err)
	}

	expected := "This is the data to send to the URL given above."
	if req.Body != expected {
		t.Errorf("incorrect body, got '%v'", req.Body)
	}
	expected = "GET"
	if req.Method != expected {
		t.Errorf("expected 'GET' method, got '%v'", req.Method)
	}
	expected = "Example org-testing---"
	if req.Name != expected {
		t.Errorf("expected '%v', got '%v'", expected, req.Name)
	}
	expected = "http://example.org/some-uri-with-dashes"
	if req.URL != expected {
		t.Errorf("expected '%v', got '%v'", expected, req.URL)
	}
	if req.Timeout != 10000 {
		t.Errorf("expected 10000, got %d", req.Timeout)
	}
	if len(req.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(req.Headers))
	}
}

// Tests a normal request, with a normal front-matter plus some body content.
func TestNewRequestNoBody(t *testing.T) {
	doc := `{
    "name": "Example org-testing---",
    "url": "http://example.org/some-uri-with-dashes",
    "method": "GET",
	"timeout": 10000,
    "headers": [
        "Authorization: Basic bG9sbGVyY29hc3Rlcgo=",
        "Some-HTTP-Header: Blah blah"
    ]
}
---`

	reader := strings.NewReader(doc)
	req, err := newRequest(reader)
	if err != nil {
		t.Errorf("failed parsing request: %v", err)
	}
	_ = req
}

// Tests a request with a front matter divider, but without any content.
func TestNewRequestNoFrontMatterWithDivider(t *testing.T) {
	doc := `---
This is the data to send to the URL given above.`

	reader := strings.NewReader(doc)
	_, err := newRequest(reader)
	if err == nil {
		t.Error("expecting error, got none")
	}
}

// Tests a request without a front-matter divider.
func TestNewRequestNoFrontMatterWithoutDivider(t *testing.T) {
	doc := `This is the data to send to the URL given above.`

	reader := strings.NewReader(doc)
	_, err := newRequest(reader)
	if err == nil {
		t.Error("expecting error, got none")
	}
}

func TestHeader(t *testing.T) {
	tests := []struct {
		Header        header
		ExpectedName  string
		ExpectedValue string
	}{
		{header("Authorization: Basic stuff"), "Authorization", "Basic stuff"},
		{header("XSS-Bla-Sed: yup!"), "XSS-Bla-Sed", "yup!"},
		{header("No-Value:"), "No-Value", ""},
	}

	g1 := header("asd")
	g1.Name()
	for _, test := range tests {
		actual := test.Header.Name()
		if actual != test.ExpectedName {
			t.Errorf("expected '%v', got '%v'", test.ExpectedName, actual)
		}
		actual = test.Header.Value()
		if actual != test.ExpectedValue {
			t.Errorf("expected '%v', got '%v'", test.ExpectedValue, actual)
		}
	}
}
