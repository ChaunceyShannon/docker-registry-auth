package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var publicDomain string
var user string
var pass string

func NewProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	return proxy, nil
}

func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var host string
		if strIn(":", r.Host) {
			host = strSplit(r.Host, ":")[0]
		} else {
			host = r.Host
		}
		lg.trace("Requests domain:", host)
		if (host == publicDomain && !itemInArray(r.Method, []string{"GET", "HEAD"})) || host != publicDomain {
			lg.trace("Need authorization")
			if err := try(func() {
				auth := r.Header.Get("Authorization")
				if !strStartsWith(auth, "Basic ") {
					panicerr("No Authorization http header")
				}
				auth = strSplit(auth)[1]
				a := strSplit(base64Decode(auth), ":")
				if a[0] != user || a[1] != pass {
					panicerr("Wrong username or password")
				}
			}).Error; err != nil {
				lg.trace("Error while checking credential:", err)
				w.Header().Set("WWW-Authenticate", "Basic realm=\"\"")
				w.WriteHeader(401)
				w.Write([]byte("Unauthorised\n"))
				return
			} else {
				lg.trace("Forward to backend server")
				proxy.ServeHTTP(w, r)
			}
		} else {
			lg.trace("Request URI:", r.RequestURI)
			if len(reFindAll("/v2/[\\-a-z0-9A-Z]+?/(blobs|manifests)/sha256:[0-9a-z]{64}", r.RequestURI)) != 0 || len(reFindAll("/v2/[\\-a-z0-9A-Z]+?/manifests/[0-9-a-zA-Z]+?$", r.RequestURI)) != 0 {
				lg.trace("Forward to backend server")
				proxy.ServeHTTP(w, r)
			} else {
				lg.trace("Forbidden")
				w.WriteHeader(403)
				return
			}
		}
	}
}

func main() {
	go func() {
		lg.trace("Start registry server with config file: /etc/docker/registry/config.yml")
		system("/entrypoint.sh /etc/docker/registry/config.yml")
	}()

	if envexists("public_domain") {
		publicDomain = getenv("public_domain")
	}
	if envexists("user") {
		user = getenv("user")
	}
	if envexists("pass") {
		pass = getenv("pass")
	}
	lg.trace("public_domain:", publicDomain)
	lg.trace("user: ", user)
	lg.trace("pass:", pass)

	proxy, err := NewProxy("http://127.0.0.1:5000")
	panicerr(err)

	http.HandleFunc("/", ProxyRequestHandler(proxy))

	lg.trace("Listen on：5001")
	log.Fatal(http.ListenAndServe(":5001", nil))
}
