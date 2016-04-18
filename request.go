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
	frontMatter := bytes.Buffer{}
	body := bytes.Buffer{}

	var writeTarget *bytes.Buffer
	writeTarget = &frontMatter

	// When the frontmatter is found, this will become true. If the frontmatter is not found,
	// this will report an error eventually.
	frontMatterFound := false

	for {
		b, err := reader.ReadByte()
		if err != nil {
			break
		}

		// Check for the separator '---'
		if b == '-' && !frontMatterFound {
			// Peek ahead at the next 2 bytes if that's '--'
			nextBytes, err := reader.Peek(2)
			if err != nil {
				return request{}, err
			}
			if string(nextBytes) == "--" {
				// Discard these two characters by advancing the reader.
				reader.Discard(2)
				// Now, check if the next two bytes is a carriage return and/or a line-feed
				nextBytes, err := reader.Peek(2)
				if err != nil {
					return request{}, err
				}
				if nextBytes[0] == '\r' || nextBytes[0] == '\n' {
					reader.Discard(1)
				}
				if nextBytes[1] == '\n' {
					reader.Discard(1)
				}

				writeTarget = &body
				frontMatterFound = true
			}

			continue
		}

		writeTarget.WriteByte(b)
	}

	if !frontMatterFound {
		return request{}, fmt.Errorf("front matter not found")
	}

	// At this point, we got some possible front matter, and a request body.
	// First try to parse the front matter.
	req := request{}
	err := json.Unmarshal(frontMatter.Bytes(), &req)
	if err != nil {
		return request{}, err
	}
	req.Body = body.String()

	return req, nil
}
