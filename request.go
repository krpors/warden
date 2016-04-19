package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// request is the struct representing a request file, with JSON front-matter
// plus the Body content.
type request struct {
	Name       string
	URL        string
	Method     string
	Timeout    int
	Headers    []header
	Assertions []assertion

	Body string
}

//============================================================================

type header string

func (h header) Name() string {
	idx := strings.Index(string(h), ":")
	if idx > 0 {
		return string(h)[0:idx]
	}

	return ""
}

func (h header) Value() string {
	idx := strings.Index(string(h), ":")
	if idx > 0 {
		return strings.Trim(string(h)[idx+1:], " ")
	}

	return ""
}

//============================================================================

// assertion is a custom string type with some methods for validating and
// executing assertions.
type assertion string

// Validate validates the assertion by compiling it, and returning the error if any.
func (a assertion) Validate() error {
	_, err := regexp.Compile(string(a))
	return fmt.Errorf("assertion regexp '%s' cannot be compiled: %v", a, err)
}

// Find compiles the assertion string and tries to find it in the given bytes content.
// Will return true if the content is found
func (a assertion) Find(content []byte) bool {
	// Validate() should be used during startup to check whether the configuration
	// file is correct. So at this point it should be good.
	re := regexp.MustCompile(string(a))
	return re.Find(content) != nil
}

//============================================================================

// newRequest attempts to build a new request object using the specified reader.
func newRequest(rd io.Reader) (request, error) {
	reader := bufio.NewReader(rd)

	var writer *bytes.Buffer            // Writer to use.
	writerFrontMatter := bytes.Buffer{} // writer for the front-matter (metadata)
	writerBody := bytes.Buffer{}        // writer for the body contents
	found := false                      // Whether the front matter was found.
	writer = &writerFrontMatter         // Start by writing to the front matter.

	for {
		b, err := reader.ReadByte()
		if err != nil {
			// no more bytes to read, stop reading by breaking
			break
		}

		writer.WriteByte(b)

		// Beginning of a new line.
		if (b == '\n' || b == '\r') && !found {
			// Peek at possible first three bytes (---)
			peek, err := reader.Peek(3)
			if err != nil {
				// Failed to peek 3 bytes, we got no three dashes.
				return request{}, fmt.Errorf("invalid file format")
			}
			if string(peek) == "---" {
				reader.Discard(3)
				// if next byte is a CR or LF, skip it.
				peek, err := reader.Peek(1)
				if err != nil {
					return request{}, err
				}
				if peek[0] == '\r' || peek[0] == '\n' {
					reader.Discard(1)
				}
				peek, err = reader.Peek(1)
				if err != nil {
					return request{}, err
				}
				if peek[0] == '\r' || peek[0] == '\n' {
					reader.Discard(1)
				}
				found = true
				writer = &writerBody
			}
		}

	}

	if !found {
		return request{}, fmt.Errorf("no front-matter found")
	}

	// At this point, we got some possible front matter, and a request body.
	// First try to parse the front matter.
	req := request{}
	err := json.Unmarshal(writerFrontMatter.Bytes(), &req)
	if err != nil {
		return request{}, err
	}
	req.Body = string(writerBody.String())

	return req, nil
}
