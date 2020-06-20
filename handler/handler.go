// Copyright © 2020 Krishna Iyer Easwaran
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package handler provides the cloud function for handling a HTTP request for Go Vanity imports.
package handler

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// config is the vanity config
type config struct {
	Host     string `yaml:"host,omitempty"`
	CacheAge *int64 `yaml:"cache_max_age,omitempty"`
	Paths    map[string]struct {
		Repo    string `yaml:"repo,omitempty"`
		Display string `yaml:"display,omitempty"`
		VCS     string `yaml:"vcs,omitempty"`
	} `yaml:"paths,omitempty"`
}

// Handler is the request handler.
type Handler struct {
	host         string
	cacheControl string
	paths        pathConfigSet
}

type pathConfigSet []pathConfig

type pathConfig struct {
	path    string
	repo    string
	display string
	vcs     string
}

// handler is the vanity imports handler.
// Making this global allows for caching of the handler state.
var handler Handler

var indexTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<h1>Welcome to {{.Host}}</h1>
<ul>
{{range .Handlers}}<li><a href="https://pkg.go.dev/{{.}}">{{.}}</a></li>{{end}}
</ul>
</html>
`))

var vanityTemplate = template.Must(template.New("vanity").Parse(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Import}} {{.VCS}} {{.Repo}}">
<meta name="go-source" content="{{.Import}} {{.Display}}">
</head>
<body>
Nothing to see here folks!
</body>
</html>`))

// client is a standard http client.
// Making this global means that we can use connection pooling to reduce the creation of a new HTTP handlers for each function invocation.
var client = &http.Client{
	Timeout: 10 * time.Second,
}

// InitHandler initializes the global handler.
// This is non-idiomatic but is optimised for google cloud functions.
// The config is parsed from a yaml file
// - Fetched from an external source (GoogleCloudStorage/Github).
func InitHandler(ctx context.Context, configURL string) error {
	// Fetch the config file from the remote path
	res, err := client.Get(configURL)
	if err != nil {
		log.Printf("Could not fetch config file: %v\n", err)
		return err
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("Could not fetch config file: %v\n", res.StatusCode)
		return fmt.Errorf("could not fetch config file: %v", res.StatusCode)
	}
	// Read out the configuration
	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Could not read config file: %v\n", err)
		return fmt.Errorf("could not read config file: %v", err)
	}
	if len(raw) == 0 {
		log.Println("Found empty config file")
		return fmt.Errorf("found empty config file")
	}

	// Parse the yaml
	var config config
	if err := yaml.Unmarshal(raw, &config); err != nil {
		log.Printf("Could not parse config: %v\n", err)
		return fmt.Errorf("could not parse config: %v", err)
	}

	handler.host = config.Host
	cacheAge := int64(86400) // 24 hours (in seconds)
	if config.CacheAge != nil {
		cacheAge = *config.CacheAge
		if cacheAge < 0 {
			return fmt.Errorf("cache_max_age is negative")
		}
	}
	handler.cacheControl = fmt.Sprintf("public, max-age=%d", cacheAge)
	for path, e := range config.Paths {
		pc := pathConfig{
			path:    strings.TrimSuffix(path, "/"),
			repo:    e.Repo,
			display: e.Display,
			vcs:     e.VCS,
		}
		switch {
		case e.Display != "":
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.display = fmt.Sprintf("%v %v/tree/master{/dir} %v/blob/master{/dir}/{file}#L{line}", e.Repo, e.Repo, e.Repo)
		case strings.HasPrefix(e.Repo, "https://bitbucket.org"):
			pc.display = fmt.Sprintf("%v %v/src/default{/dir} %v/src/default{/dir}/{file}#{file}-{line}", e.Repo, e.Repo, e.Repo)
		}
		switch {
		case e.VCS != "":
			if e.VCS != "bzr" && e.VCS != "git" && e.VCS != "hg" && e.VCS != "svn" {
				return fmt.Errorf("configuration for %v: unknown VCS %s", path, e.VCS)
			}
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.vcs = "git"
		default:
			return fmt.Errorf("configuration for %v: cannot infer VCS from %s", path, e.Repo)
		}
		handler.paths = append(handler.paths, pc)
	}
	sort.Sort(handler.paths)
	return nil
}

// init runs only on a cold start
func init() {
	InitHandler(context.Background(), os.Getenv("CONFIG_URL"))
}

// HandleImport handles Go's vanity import requests.
func HandleImport(w http.ResponseWriter, r *http.Request) {
	current := r.URL.Path
	pc, subpath := handler.paths.find(current)
	if pc == nil && current == "/" {
		handler.serveIndex(w, r)
		return
	}
	if pc == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", handler.cacheControl)
	if err := vanityTemplate.Execute(w, struct {
		Import  string
		Subpath string
		Repo    string
		Display string
		VCS     string
	}{
		Import:  handler.Host(r) + pc.path,
		Subpath: subpath,
		Repo:    pc.repo,
		Display: pc.display,
		VCS:     pc.vcs,
	}); err != nil {
		http.Error(w, "cannot render the page", http.StatusInternalServerError)
	}
}

// serveIndex serves the list of all supported paths for this host.
func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	host := h.Host(r)
	handlers := make([]string, len(h.paths))
	for i, h := range h.paths {
		handlers[i] = host + h.path
	}
	if err := indexTemplate.Execute(w, struct {
		Host     string
		Handlers []string
	}{
		Host:     host,
		Handlers: handlers,
	}); err != nil {
		http.Error(w, "cannot render the page", http.StatusInternalServerError)
	}
}

// Host returns a the host.
func (h *Handler) Host(r *http.Request) string {
	host := h.host
	if host == "" {
		return r.Host
	}
	return host
}

func (pset pathConfigSet) Len() int {
	return len(pset)
}

func (pset pathConfigSet) Less(i, j int) bool {
	return pset[i].path < pset[j].path
}

func (pset pathConfigSet) Swap(i, j int) {
	pset[i], pset[j] = pset[j], pset[i]
}

func (pset pathConfigSet) find(path string) (pc *pathConfig, subpath string) {
	// Fast path with binary search to retrieve exact matches
	// e.g. given pset ["/", "/abc", "/xyz"], path "/def" won't match.
	i := sort.Search(len(pset), func(i int) bool {
		return pset[i].path >= path
	})
	if i < len(pset) && pset[i].path == path {
		return &pset[i], ""
	}
	if i > 0 && strings.HasPrefix(path, pset[i-1].path+"/") {
		return &pset[i-1], path[len(pset[i-1].path)+1:]
	}

	// Slow path, now looking for the longest prefix/shortest subpath i.e.
	// e.g. given pset ["/", "/abc/", "/abc/def/", "/xyz"/]
	//  * query "/abc/foo" returns "/abc/" with a subpath of "foo"
	//  * query "/x" returns "/" with a subpath of "x"
	lenShortestSubpath := len(path)
	var bestMatchConfig *pathConfig

	// After binary search with the >= lexicographic comparison,
	// nothing greater than i will be a prefix of path.
	max := i
	for i := 0; i < max; i++ {
		ps := pset[i]
		if len(ps.path) >= len(path) {
			// We previously didn't find the path by search, so any
			// route with equal or greater length is NOT a match.
			continue
		}
		sSubpath := strings.TrimPrefix(path, ps.path)
		if len(sSubpath) < lenShortestSubpath {
			subpath = sSubpath
			lenShortestSubpath = len(sSubpath)
			bestMatchConfig = &pset[i]
		}
	}
	return bestMatchConfig, subpath
}
