package tritonhttp

//TODO
// 	Date
// ○ Last-Modified (required only if return type is 200)
// ○ Content-Type (required only if return type is 200)
// ○ Content-Length (required only if return type is 200)
// ○ Connection: close (returned in response to a client “Connection: close” header, or
// for a 400 response)

// Our response messages might have a body if it’s a 200 response. In this case, the message
// body is basically the bytes of the requested file to serve to the client. 400 and 404 messages
// don’t have a body.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// VirtualHosts contains a mapping from host name to the docRoot path
	// (i.e. the path to the directory to serve static files from) for
	// all virtual hosts that this server supports
	//TODO
	VirtualHosts map[string]string // DocRoot string in discussion
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.

func (s *Server) ListenAndServe() error {

	// Hint: Validate all docRoots
	// Hint: create your listen socket and spawn off goroutines per incoming client

	// Validate the configuration of the server
	if err := s.ValidateServerSetup(); err != nil {
		return err
	}
	fmt.Println("Server setup valid!")

	// server should now start to listen on the configured address
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	fmt.Println("Listening on", ln.Addr())

	// making sure the listener is closed when we exit
	defer func() {
		err = ln.Close()
		if err != nil {
			fmt.Println("error in closing listener", err)
		}
	}() //TODO why there is an extra() not defer.func(){}()

	// accept connections forever
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		fmt.Println("accepted connection", conn.RemoteAddr())
		go s.HandleConnection(conn)
	}
}

func (s *Server) ValidateServerSetup() error {
	// Validating the doc root of the server
	for _, value := range s.VirtualHosts {
		fi, err := os.Stat(value)
		if os.IsNotExist(err) {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("doc root %q is not a directory", value)
		}

	}
	return nil

	// fi, err := os.Stat(s.DocRoot)
	// if os.IsNotExist(err) {
	// 	return err
	// }
	// if !fi.IsDir() {
	// 	return fmt.Errorf("doc root %q is not a directory", s.DocRoot)
	// }
	// return nil
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	br := bufio.NewReader(conn)

	for {
		// Set timeout
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Printf("Failed to set timeout for connection %v", conn)
			_ = conn.Close()
			return
		}

		// Read next request from the client
		req, MoreBytes, err := ReadRequest(br)

		// Handle EOF
		if errors.Is(err, io.EOF) {
			log.Printf("Connection closed by %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}

		// timeout in this application means we just close the connection
		// Note : proj3 might require you to do a bit more here
		if err, ok := err.(net.Error); ok && err.Timeout() {
			if !MoreBytes {
				log.Printf("Connection to %v timed out", conn.RemoteAddr())
				_ = conn.Close()
				return
			}
			res := &Response{}
			res.HandleBadRequest()
			//TODO 这是啥意思
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}

		// Handle the request which is not a GET and immediately close the connection and return
		if err != nil {
			log.Printf("Handle bad request for error: %v", err)
			res := &Response{}
			res.HandleBadRequest()
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}

		// Handle good request
		log.Printf("Handle good request: %v", req)
		res := s.HandleGoodRequest(req)
		err = res.Write(conn)
		if err != nil {
			fmt.Println(err)
		}

		// We'll never close the connection and handle as many requests for this connection and pass on this
		// responsibility to the timeout mechanism
		if req.Close {
			_ = conn.Close() //TODO why??
			return
		}
	}
}

func (s *Server) HandleGoodRequest(req *Request) (res *Response) {
	res = &Response{}
	res.init(req)
	absolutePath := filepath.Join(s.VirtualHosts[req.Host], req.URL)
	absolutePath = filepath.Clean(absolutePath)

	if absolutePath[:len(s.VirtualHosts[req.Host])] != s.VirtualHosts[req.Host] {
		res.HandleNotFound()
	} else {
		res.HandleOK(req, absolutePath)
	}

	return res
}

func (res *Response) HandleNotFound() {
	res.StatusCode = statusNotFound
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path string) {
	// res.init(req)
	res.FilePath = path
	res.StatusCode = statusOK
	stats, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Print(err)
		res.HandleNotFound()
		return
	}
	res.Headers["Last-Modified"] = FormatTime(stats.ModTime())
	res.Headers["Content-Type"] = MIMETypeByExtension(filepath.Ext(path))
	res.Headers["Content-Length"] = strconv.FormatInt(stats.Size(), 10)
}

// HandleBadRequest prepares res to be a 405 Method Not allowed response
func (res *Response) HandleBadRequest() {
	res.init(nil)
	res.StatusCode = statusBadRequest
	res.FilePath = ""
	res.Request = nil
	res.Headers["Connection"] = "close"
}

func (res *Response) init(req *Request) {
	res.Proto = responseProto
	res.Headers = make(map[string]string)
	res.Headers["Date"] = FormatTime(time.Now())
	if req != nil {
		if req.URL[len(req.URL)-1] == '/' {
			req.URL = req.URL + "index.html"
		}
		if req.Close {
			res.Headers["Connection"] = "close"
		}
	}
}
