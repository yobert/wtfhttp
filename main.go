package main

import (
	"bytes"
	"flag"
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

var (
	maxbody       = 1024
	maxheader     = 64
	maxheadertail = 24
	serialize     = false
	shortheaders  = false
)

var shortheaderlist = map[string]bool{
	"Accept":                            true,
	"Accept-Encoding":                   true,
	"Accept-Language":                   true,
	"Access-Control-Allow-Methods":      true,
	"Access-Control-Allow-Origin":       true,
	"Access-Control-Expose-Headers":     true,
	"Access-Control-Max-Age":            true,
	"Cache-Control":                     true,
	"Connection":                        true,
	"Dnt":                               true,
	"Etag":                              true,
	"If-None-Match":                     true,
	"Origin":                            true,
	"Referer":                           true,
	"Referrer-Policy":                   true,
	"Sec-Fetch-Dest":                    true,
	"Sec-Fetch-Mode":                    true,
	"Sec-Fetch-Site":                    true,
	"Server-Timing":                     true,
	"User-Agent":                        true,
	"Vary":                              true,
	"X-Content-Type-Options":            true,
	"X-Download-Options":                true,
	"X-Forwarded-For":                   true,
	"X-Forwarded-Proto":                 true,
	"X-Frame-Options":                   true,
	"X-Permitted-Cross-Domain-Policies": true,
	"X-Runtime":                         true,
	"X-Xss-Protection":                  true,
}

func main() {
	flag.BoolVar(&shortheaders, "short", false, "Hide unimportant headers")
	flag.BoolVar(&serialize, "serialize", false, "Force serialization")
	flag.IntVar(&maxbody, "maxbody", 1024, "Max body bytes to display")
	flag.Parse()

	listen := ":8080"
	target := "localhost:80"
	target_scheme := "http"

	args := flag.Args()

	if len(args) > 0 {
		listen = args[0]
	}
	if len(args) > 1 {
		target = args[1]
	}

	if strings.HasPrefix(target, "https://") {
		target = strings.TrimPrefix(target, "https://")
		target_scheme = "https"
	} else if strings.HasPrefix(target, "http://") {
		target = strings.TrimPrefix(target, "http://")
		target_scheme = "http"
	}

	colors := terminal.IsTerminal(int(os.Stdout.Fd()))

	log.Printf("Listening on %s, proxying requests to %s://%s, colors %#v, serialize %#v shortheaders %#v\n", listen, target_scheme, target, colors, serialize, shortheaders)

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

			req.URL.Scheme = target_scheme
			req.URL.Host = target
			req.Host = target

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
		if shortheaders && shortheaderlist[k] {
			continue
		}

		for _, v := range headers[k] {
			print_header(dest, colors, clr, k, v)
		}
		for _, v := range extra[k] {
			print_header(dest, colors, clr, k, v)
		}
	}
}

func print_header(dest io.Writer, colors bool, clr Color, k, v string) {
	if shortheaders && len(v) > maxheader {
		v = v[:maxheader-maxheadertail] + " ‥ " + v[len(v)-maxheadertail:]
	}
	if colors {
		fmt.Fprintln(dest, BrightText(clr)+k+Reset()+":", Text(clr)+v+Reset())
	} else {
		fmt.Fprintln(dest, k+":", v)
	}
}

func print_mime(dest io.Writer, colors bool, clr Color, buf []byte) {
	if len(buf) == 0 {
		return
	}
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
