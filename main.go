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
				// read all bytes from content body and create new stream using it.
				bodyBytes, _ := ioutil.ReadAll(req.Body)
				req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

				print_body(bodyBytes)

				// create new request for parsing the body
				//req2, _ := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(bodyBytes))
				//req2.Header = req.Header
				//req2.ParseForm()
				//log.Println(req2.Form)
			}

			req.URL.Scheme = "http"
			req.URL.Host = target
			//req.Header.Set("Host", "pkunk.org")

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
	escaped := string(buf)
	escaped = strconv.Quote(escaped)
	escaped = escaped[1 : len(escaped)-1]
	escaped = strings.Replace(escaped, "\\n", "\n", -1)
	escaped = strings.Replace(escaped, "\\\"", "\"", -1)
	fmt.Println(escaped)
}
