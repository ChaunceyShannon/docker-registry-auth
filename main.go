package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	. "github.com/ChaunceyShannon/golanglibs"
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
		if String(":").In(r.Host) {
			host = String(r.Host).Split(":")[0].S
		} else {
			host = r.Host
		}
		Lg.Trace("Requests domain:", host)
		if (host == publicDomain && !Array([]string{"GET", "HEAD"}).Has(r.Method)) || host != publicDomain {
			Lg.Trace("Need authorization")
			if err := Try(func() {
				auth := r.Header.Get("Authorization")
				if !String(auth).StartsWith("Basic ") {
					Panicerr("No Authorization http header")
				}
				auth = String(auth).Split()[1].S
				a := String(Base64.Decode(auth)).Split(":")
				if a[0].S != user || a[1].S != pass {
					Panicerr("Wrong username or password")
				}
			}).Error; err != nil {
				Lg.Trace("Error while checking credential:", err)
				w.Header().Set("WWW-Authenticate", "Basic realm=\"\"")
				w.WriteHeader(401)
				w.Write([]byte("Unauthorised\n"))
				return
			} else {
				Lg.Trace("Forward to backend server")
				proxy.ServeHTTP(w, r)
			}
		} else if r.RequestURI == "/v2" || r.RequestURI == "/v2/" { // Anchore need to access this
			w.WriteHeader(200)
			w.Write([]byte("{}"))
			return
		} else {
			Lg.Trace("Request URI:", r.RequestURI)
			if len(Re.FindAll("/v2/[a-z0-9A-Z]+?/(blobs|manifests)/sha256:[0-9a-z]{64}", r.RequestURI)) != 0 || len(Re.FindAll("/v2/[a-z0-9A-Z]+?/manifests/[0-9-a-zA-Z]+?$", r.RequestURI)) != 0 {
				Lg.Trace("Forward to backend server")
				proxy.ServeHTTP(w, r)
			} else {
				Lg.Trace("Forbidden")
				w.WriteHeader(403)
				return
			}
		}
	}
}

func main() {
	go func() {
		Lg.Trace("Start registry server with config file: /etc/docker/registry/config.yml")
		Os.System("/entrypoint.sh /etc/docker/registry/config.yml")
	}()

	if Os.Envexists("public_domain") {
		publicDomain = Os.Getenv("public_domain")
	}
	if Os.Envexists("user") {
		user = Os.Getenv("user")
	}
	if Os.Envexists("pass") {
		pass = Os.Getenv("pass")
	}
	Lg.Trace("public_domain:", publicDomain)
	Lg.Trace("user: ", user)
	Lg.Trace("pass:", pass)

	proxy, err := NewProxy("http://127.0.0.1:5000")
	Panicerr(err)

	http.HandleFunc("/", ProxyRequestHandler(proxy))

	Lg.Trace("Listen on：5001")
	log.Fatal(http.ListenAndServe(":5001", nil))
}
