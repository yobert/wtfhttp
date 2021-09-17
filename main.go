package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gabriel-vasile/mimetype"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	maxbody   = 1024
	serialize = false
)

func main() {
	listen := ":8080"
	target := "localhost:80"

	if len(os.Args) > 1 {
		listen = os.Args[1]
	}
	if len(os.Args) > 2 {
		target = os.Args[2]
	}

	colors := terminal.IsTerminal(int(os.Stdout.Fd()))

	log.Printf("Listening on %s, proxying requests to %s, colors %#v, serialize %#v\n", listen, target, colors, serialize)

	var nextid uint64

	var lock sync.Mutex

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			if serialize {
				lock.Lock()
			}

			id := atomic.AddUint64(&nextid, 1)

			o := &bytes.Buffer{}

			if colors {
				fmt.Fprintln(o, All(Black, false, Blue)+fmt.Sprintf(" %d ", id)+Reset(), BrightText(Blue)+req.Method+Reset(), Text(Blue)+req.URL.String()+Reset())
			} else {
				fmt.Fprintln(o, id, req.Method, req.URL)
			}
			print_headers(o, colors, req.Header, Blue, map[string][]string{"Host": {req.Host}})
			fmt.Fprintln(o)

			if req.Body != nil {
				bodyBytes, err := ioutil.ReadAll(req.Body)
				if err != nil {
					log.Println(err)
				} else {
					print_mime(o, colors, Blue, bodyBytes)
					req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
					print_body(o, bodyBytes)
				}
			}

			req.URL.Scheme = "http"
			req.URL.Host = target

			req.Header.Set("X-WTF-ID", fmt.Sprintf("%d", id))

			fmt.Print(o.String())
		},
		ModifyResponse: func(res *http.Response) error {
			id, _ := strconv.Atoi(res.Request.Header.Get("X-WTF-ID"))

			c := Green
			if res.StatusCode >= 400 {
				c = Red
			}

			o := &bytes.Buffer{}

			if colors {
				fmt.Fprintln(o, All(Black, false, c)+fmt.Sprintf(" %d ", id)+Reset(), BrightText(c)+res.Status+Reset())
			} else {
				fmt.Fprintln(o, id, res.Status)
			}
			print_headers(o, colors, res.Header, c, map[string][]string{})
			fmt.Fprintln(o)

			if res.Body != nil {
				bodyBytes, _ := ioutil.ReadAll(res.Body)
				print_mime(o, colors, c, bodyBytes)
				res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				print_body(o, bodyBytes)
			}

			fmt.Print(o.String())

			if serialize {
				lock.Unlock()
			}

			return nil
		},
	}

	log.Fatal(http.ListenAndServe(listen, proxy))
}

func print_headers(dest io.Writer, colors bool, headers http.Header, clr Color, extra map[string][]string) {
	keys := make([]string, 0, len(headers))
	for k, _ := range headers {
		keys = append(keys, k)
	}
	for k, _ := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range headers[k] {
			if colors {
				fmt.Fprintln(dest, BrightText(clr)+k+Reset()+":", Text(clr)+v+Reset())
			} else {
				fmt.Fprintln(dest, k+":", v)
			}
		}
		for _, v := range extra[k] {
			if colors {
				fmt.Fprintln(dest, BrightText(clr)+k+Reset()+":", Text(clr)+v+Reset())
			} else {
				fmt.Fprintln(dest, k+":", v)
			}
		}
	}
}

func print_mime(dest io.Writer, colors bool, clr Color, buf []byte) {
	mtype := mimetype.Detect(buf)
	if mtype.String() == "application/octet-stream" {
		return
	}
	if colors {
		fmt.Fprintln(dest, Text(clr)+mtype.String()+Reset())
	} else {
		fmt.Fprintln(dest, mtype)
	}
	fmt.Fprintln(dest)
}

func print_body(dest io.Writer, buf []byte) {
	if len(buf) == 0 {
		return
	}

	trimmed := 0
	if len(buf) > maxbody {
		trimmed = len(buf) - maxbody
		buf = buf[0:maxbody]
	}
	escaped := string(buf)
	escaped = strconv.Quote(escaped)
	escaped = escaped[1 : len(escaped)-1]
	escaped = strings.Replace(escaped, "\\n", "\n", -1)
	escaped = strings.Replace(escaped, "\\\"", "\"", -1)
	if trimmed > 0 {
		escaped += fmt.Sprintf(" (trimmed %d bytes)", trimmed)
	}
	fmt.Fprintln(dest, strings.TrimSpace(escaped))
	fmt.Fprintln(dest)
}
