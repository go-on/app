package files

import (
	"fmt"
	"github.com/go-on/app"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func noDirListing(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

type Files struct {
	mountPath  app.MountPath
	fsDir      string
	fileServer http.Handler
}

func New(mountPath app.MountPath, dir string) *Files {
	var p = &Files{}
	f, err := os.Stat(dir)
	if err != nil {
		panic(fmt.Sprintf("can't create fileserver for %#v: directory does not exist", dir))
	}

	if !f.IsDir() {
		panic(fmt.Sprintf("can't create fileserver for %#v: no directory", dir))
	}
	p.fsDir = dir
	p.mountPath = mountPath
	p.fileServer = noDirListing(http.FileServer(http.Dir(dir)))
	return p
}

// if the file does not exist or is a directory, path is an empty string
func (f *Files) Link(relPath string) (l app.AppLink) {
	l.Method = app.GET

	// fmt.Printf("path: %#v\n", filepath.Join(f.fsDir, relPath))
	fi, err := os.Stat(filepath.Join(f.fsDir, relPath))
	if err != nil {
		l.Path = "#file-does-not-exist"
		return
	}
	if fi.IsDir() {
		l.Path = "#is-directory"
		return
	}
	l.Path = "/" + filepath.Join(string(f.mountPath), relPath)
	return
}

func (a *Files) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	a.fileServer.ServeHTTP(rw, r)
}

func (p Files) MountPath() app.MountPath {
	return p.mountPath
}

func (p *Files) Protected(authorizer app.Authorizer, authenticator app.Authenticator) app.App {
	var pt = &protected{}
	pt.Files = p
	pt.authorizer = authorizer
	pt.authenticator = authenticator
	return pt
}

type protected struct {
	*Files
	authorizer    app.Authorizer
	authenticator app.Authenticator
}

func (a *protected) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	user := a.authenticator.Authenticate(r)
	if user == nil {
		rw.WriteHeader(http.StatusForbidden)
		return
	}
	user.Authorizer = a.authorizer
	path := app.EndPath{app.Method(r), r.URL.Path}
	if !user.Authorize(path) {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	a.Files.ServeHTTP(rw, r)
}

var _ app.App = &Files{}
var _ app.App = &protected{}
