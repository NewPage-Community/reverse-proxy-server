package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/cssivision/reverseproxy"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	remote *url.URL
	endpoint *url.URL
)

func main() {
	httpProxy, err := url.Parse(os.Getenv("HTTP_PROXY"))
	if err != nil {
		panic(err)
	}
	remote, err = url.Parse(os.Getenv("REVERSE_PROXY_REMOTE"))
	if err != nil {
		panic(err)
	}
	endpoint, err = url.Parse(os.Getenv("REVERSE_PROXY_ENDPOINT"))
	if err != nil {
		panic(err)
	}

	proxy := reverseproxy.NewReverseProxy(remote)
	proxy.Transport = &http.Transport{Proxy: http.ProxyURL(httpProxy)}
	proxy.ModifyResponse = rewrite
	http.HandleFunc("/", handler(proxy))
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func handler(p *reverseproxy.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
}

func rewrite(r *http.Response) (err error) {
	if r.StatusCode == http.StatusOK {
		var buf []byte
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				return err
			}
			defer gr.Close()
			buf, err = ioutil.ReadAll(gr)
			if err != nil {
				return err
			}
			r.Header.Del("Content-Encoding")
		} else {
			buf, err = ioutil.ReadAll(r.Body)
			if err != nil {
				return err
			}
		}
		if buf != nil {
			if endpoint.Scheme == "http" {
				buf = bytes.Replace(buf, []byte("https://"+remote.Host), []byte("http://"+endpoint.Host), -1)
			} else {
				buf = bytes.Replace(buf, []byte(remote.Host), []byte(endpoint.Host), -1)
			}
			r.Body = ioutil.NopCloser(bytes.NewReader(buf))
			r.ContentLength = int64(len(buf))
			r.Header.Set("Content-Length", fmt.Sprint(len(buf)))
		}
	}
	return nil
}