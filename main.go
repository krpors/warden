package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const version = "1.0.0"

// noopWriter is a writer which write to nowhere.
// TODO: isn't there a builtin type for this? Can't seem to find it.
type noopWriter struct {
}

// Write writes no bytes, adn returns 0 bytes written and no error.
func (n *noopWriter) Write(p []byte) (int, error) {
	return 0, nil
}

// Program flags.
var (
	flagDebug = flag.Bool("debug", false, "enable debugging/verbosity")
	flagDir   = flag.String("dir", ".", "directory with request files")
)

var (
	debug = log.New(&noopWriter{}, "DEBUG ", log.LstdFlags)
)

type result struct {
	Request      request       // The initial request information to send.
	Response     string        // response as a string
	ResponseTime time.Duration // response time
	Error        error         // possible error
}

// String returns a string representation of the result.
func (r result) String() string {
	b := &bytes.Buffer{}
	if r.Error == nil {
		fmt.Fprint(b, "OK    ")
	} else {
		fmt.Fprint(b, "FAIL  ")
	}

	fmt.Fprintf(b, "%s (%d ms)", r.Request.Name, r.ResponseTime/time.Millisecond)

	if r.Error != nil {
		fmt.Fprintf(b, "; error: %v", r.Error)
	}

	return b.String()
}

// scanDirectory scans a directory for request configuration files. Will return
// a slice of request objects, or a non-nil error if anything fails.
func scanDirectory(d string) ([]request, error) {
	debug.Printf("Scanning directory '%s'\n", d)

	var requests []request

	finfo, err := os.Stat(d)
	if err != nil {
		return nil, fmt.Errorf("unable to stat directory '%s'", d)
	}

	if !finfo.IsDir() {
		return nil, fmt.Errorf("'%s' is not a directory", d)
	}

	dir, err := os.Open(d)
	if err != nil {
		return nil, err
	}

	finfos, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}

	for _, fileInfo := range finfos {
		if fileInfo.IsDir() {
			debug.Printf("Skipping directory '%s/%s'\n", d, fileInfo.Name())
			continue
		}

		filepath := path.Join(d, fileInfo.Name())
		file, err := os.Open(filepath)
		if err != nil {
			// TODO: mark as failure.
			fmt.Println(err)
			continue
		}
		defer file.Close()

		req, err := newRequest(file)
		if err != nil {
			debug.Printf("Could not parse request file '%s': %v", fileInfo.Name(), err)
			continue
		}

		requests = append(requests, req)
	}

	debug.Printf("Found %d correct requests", len(requests))

	return requests, nil
}

// send sends an HTTP request using the given request object r. The result is sent
// to the given channel c.
func send(r request, c chan result) {
	client := http.Client{}
	reader := strings.NewReader(r.Body)
	req, err := http.NewRequest(r.Method, r.URL, reader)
	if err != nil {
		c <- result{r, "", -1, err}
		return
	}

	// This block enables us to timeout the HTTP call.
	type response struct {
		Resp *http.Response
		Err  error
	}
	timeoutChan := make(chan response, 1)

	tstart := time.Now()
	go func() {
		r, err := client.Do(req)
		timeoutChan <- response{r, err}
	}()

	timeout := time.Duration(r.Timeout) * time.Millisecond

	var theResponse response

	select {
	case <-time.After(timeout):
		c <- result{r, "", -1, fmt.Errorf("timeout after %d ms", r.Timeout)}
		return
	case theResponse = <-timeoutChan:
	}

	responseTime := time.Now().Sub(tstart)

	if theResponse.Err != nil {
		c <- result{r, "", -1, theResponse.Err}
		return
	}

	str, err := ioutil.ReadAll(theResponse.Resp.Body)
	if err != nil {
		c <- result{r, "", -1, err}
		return
	}

	// Print some debugging information, if applicable.
	if *flagDebug {
		debug.Printf("[%s]: HTTP request:\n%s", r.Name, r.Body)
		for _, k := range r.Headers {
			debug.Printf("[%s]: HTTP request header: %s\n", r.Name, k)
		}
		for k, v := range theResponse.Resp.Header {
			debug.Printf("[%s]: HTTP response header: %s=%s\n", r.Name, k, v[0])
		}
		debug.Printf("[%s]: HTTP response:\n%s", r.Name, str)
	}

	for _, assert := range r.Assertions {
		if !assert.Find(str) {
			c <- result{r, "", responseTime, fmt.Errorf("assertion failed: '%s'", assert)}
			return
		}
	}

	// Result ok, assertions matched, no error.
	c <- result{r, string(str), responseTime, nil}
}

// run iterates over the requests, sends them to their destinations. Gather results.
func run(requests []request) error {
	c := make(chan result)
	for _, request := range requests {
		go send(request, c)
	}

	for range requests {
		res := <-c
		fmt.Println(res)
	}

	return nil
}

// usage prints the usage of the program.
func usage() {
	fmt.Fprintf(os.Stderr, "warden version %s\n", version)
	fmt.Fprintf(os.Stderr, `
TODO: some explanation.

FLAGS (with defaults):
`)
	flag.PrintDefaults()
	os.Exit(4)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *flagDebug {
		debug.SetOutput(os.Stdout)
	}

	requests, err := scanDirectory(*flagDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to scan directory '%s': %v\n", *flagDir, err)
		os.Exit(3)
	}

	run(requests)
}
