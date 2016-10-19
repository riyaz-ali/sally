package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// CreateHandler creates a Sally http.Handler
func CreateHandler(config Config) *httprouter.Router {
	router := httprouter.New()
	router.RedirectTrailingSlash = false
	router.NotFound = notFoundHandlerFunc
	router.HandleMethodNotAllowed = true
	router.MethodNotAllowed = methodNotAllowedHandlerFunc
	router.PanicHandler = panicHandlerFunc

	router.GET("/", indexHandler{config: config}.Handle)

	for name, pkg := range config.Packages {
		handle := packageHandler{
			pkgName: name,
			pkg:     pkg,
			config:  config,
		}.Handle
		router.GET(fmt.Sprintf("/%s", name), handle)
		router.GET(fmt.Sprintf("/%s/*path", name), handle)
	}

	return router
}

type indexHandler struct {
	config Config
}

func (h indexHandler) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if err := indexTemplate.Execute(w, h.config); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

var indexTemplate = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html>
    <body>
        <ul>
            {{ range $key, $value := .Packages }}
	  	        <li>{{ $key }} - {{ $value.Repo }}</li>
	        {{ end }}
        </ul>
    </body>
</html>
`))

type packageHandler struct {
	pkgName string
	pkg     Package
	config  Config
}

func (h packageHandler) Handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	canonicalURL := fmt.Sprintf("%s/%s", h.config.URL, h.pkgName)
	data := struct {
		Repo         string
		CanonicalURL string
		GodocURL     string
	}{
		Repo:         h.pkg.Repo,
		CanonicalURL: canonicalURL,
		GodocURL:     fmt.Sprintf("https://godoc.org/%s%s", canonicalURL, ps.ByName("path")),
	}
	if err := packageTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

var packageTemplate = template.Must(template.New("package").Parse(`
<!DOCTYPE html>
<html>
    <head>
        <meta name="go-import" content="{{ .CanonicalURL }} git https://{{ .Repo }}">
        <meta name="go-source" content="{{ .CanonicalURL }} https://{{ .Repo }} https://{{ .Repo }}/tree/master{/dir} https://{{ .Repo }}/tree/master{/dir}/{file}#L{line}">
        <meta http-equiv="refresh" content="0; url={{ .GodocURL }}">
    </head>
    <body>
        Nothing to see here. Please <a href="{{ .GodocURL }}">move along</a>.
    </body>
</html>
`))

func notFoundHandlerFunc(w http.ResponseWriter, r *http.Request) {
	// don't cache 404s so that edge caches always request from origin
	w.Header().Set("Cache-Control", "no-cache")

	w.WriteHeader(http.StatusNotFound)
	io.WriteString(w, "404 page not found")
}

func methodNotAllowedHandlerFunc(w http.ResponseWriter, r *http.Request) {
	// don't cache 405s so that edge caches always request from origin
	w.Header().Set("Cache-Control", "no-cache")

	w.WriteHeader(http.StatusMethodNotAllowed)
	io.WriteString(w, "405 method not allowed")
}

func panicHandlerFunc(w http.ResponseWriter, r *http.Request, i interface{}) {
	// don't cache 500s so that edge caches always request from origin
	w.Header().Set("Cache-Control", "no-cache")

	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, "500 internal server error")

	// TODO write error to log
}