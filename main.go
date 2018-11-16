// Copyright (c) 2018 Noel Cower
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Command fsv is a very simple file-server. It is intended only for testing and should not be
// relied upon in any sort of production environment.
package main

import (
	"flag"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
)

var allowedPaths = map[string]string{}
var prefix = "/"

func main() {
	addr := flag.String("listen", "127.0.0.1:8080", "Listen address")
	fprefix := flag.String("prefix", "/", "Prefix to expect and strip from paths.")
	flag.CommandLine.Parse(append([]string{"-logtostderr"}, os.Args[1:]...))

	prefix = *fprefix
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	for _, p := range flag.Args() {
		var accept string
		if sep := strings.IndexByte(p, ':'); sep != -1 {
			p, accept = p[:sep], p[sep+1:]
		} else {
			accept = path.Base(p)
		}

		for _, sub := range strings.Split(accept, ",") {
			if sub == prefix {
				// Nop
			} else if strings.HasSuffix(prefix, "/") && sub == "/" {
				sub = prefix
			} else {
				sub = prefix + sub
			}

			allowedPaths[sub] = p
			glog.Infof("Serving %q from %q", p, sub)
		}
	}

	http.ListenAndServe(*addr, http.HandlerFunc(serve))
}

type responseCoder struct {
	written bool
	code    int
	http.ResponseWriter
}

func (r *responseCoder) WriteHeader(code int) {
	if !r.written {
		r.code = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCoder) Write(p []byte) (int, error) {
	if !r.written {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(p)
}

func serve(w http.ResponseWriter, r *http.Request) {
	if glog.V(1) {
		rc := &responseCoder{ResponseWriter: w}
		w = rc
		defer func() {
			glog.Infof("%d %q %q %q %q", rc.code, r.RemoteAddr, r.Header["X-Forwarded-For"], r.Method, r.URL)
		}()
	}
	fp, ok := allowedPaths[r.URL.Path]
	if !ok {
		glog.Infof("404\t%q\t%q", r.Method, r.URL)
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, fp)
}
