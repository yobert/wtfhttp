package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"sort"
	"strconv"
	"strings"
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

	log.Printf("Listening on %s, proxying requests to %s\n", listen, target)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			log.Println(req.Proto, req.Method, req.URL)
			print_headers(req.Header)
			fmt.Println("Host:", req.Host)
			fmt.Println()

			if req.Body != nil {
				bodyBytes, err := ioutil.ReadAll(req.Body)
				if err != nil {
					log.Println(err)
				} else {
					req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
					print_body(bodyBytes)
				}
			}

			req.URL.Scheme = "http"
			req.URL.Host = target
		},
		ModifyResponse: func(res *http.Response) error {
			fmt.Println(res.Proto, res.Status)
			print_headers(res.Header)
			fmt.Println()

			if res.Body != nil {
				bodyBytes, _ := ioutil.ReadAll(res.Body)
				res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				print_body(bodyBytes)
			}

			return nil
		},
	}

	log.Fatal(http.ListenAndServe(listen, proxy))
}

func print_headers(headers http.Header) {
	keys := make([]string, 0, len(headers))
	for k, _ := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range headers[k] {
			fmt.Println(k+":", v)
		}
	}
}

func print_body(buf []byte) {
	trimmed := 0
	if len(buf) > 8192 {
		trimmed = len(buf) - 8192
		buf = buf[0:8192]
	}
	escaped := string(buf)
	escaped = strconv.Quote(escaped)
	escaped = escaped[1 : len(escaped)-1]
	escaped = strings.Replace(escaped, "\\n", "\n", -1)
	escaped = strings.Replace(escaped, "\\\"", "\"", -1)
	if trimmed > 0 {
		escaped += fmt.Sprintf(" (trimmed %d bytes)", trimmed)
	}
	fmt.Println(escaped)
}
