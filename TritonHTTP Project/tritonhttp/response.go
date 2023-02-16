package tritonhttp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
)

type Response struct {
	Proto      string // e.g. "HTTP/1.1"
	StatusCode int    // e.g. 200
	StatusText string // e.g. "OK"

	// Headers stores all headers to write to the response.
	Headers map[string]string

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	// Hint: you might need this to handle the "Connection: Close" requirement
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

const (
	responseProto = "HTTP/1.1"

	statusOK         = 200
	statusNotFound   = 404
	statusBadRequest = 400
)

var statusText = map[int]string{
	statusOK:         "OK",
	statusNotFound:   "Not Found",
	statusBadRequest: "Bad Request",
}

func (res *Response) Write(w io.Writer) error {
	bw := bufio.NewWriter(w)

	statusLine := fmt.Sprintf("%v %v %v\r\n", res.Proto, res.StatusCode, statusText[res.StatusCode])
	if _, err := bw.WriteString(statusLine); err != nil {
		return err
	}
	if err := res.WriteHeaders(bw); err != nil {
		return err
	}
	if err := res.WriteBody(bw); err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

func (res *Response) WriteHeaders(w io.Writer) error {
	bw := bufio.NewWriter(w)
	keys := make([]string, 0, len(res.Headers))
	for key := range res.Headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		header := key + ": " + res.Headers[key] + "\r\n"
		if _, err := bw.Write([]byte(header)); err != nil {
			return err
		}
	}
	if _, err := bw.Write([]byte("\r\n")); err != nil {
		return err
	}

	return nil
}

func (res *Response) WriteBody(w io.Writer) error {
	bw := bufio.NewWriter(w)
	var body []byte
	var err error
	if len(res.FilePath) > 0 {
		if body, err = os.ReadFile(res.FilePath); err != err {
			return err
		}
	}
	if _, err := bw.Write(body); res != nil {
		return err
	}

	return nil
}
