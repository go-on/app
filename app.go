package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

type method string
type Status int

func (m method) String() string {
	return string(m)
}

func Method(r *http.Request) method {
	switch r.Method {
	case "GET":
		return GET
	case "POST":
		return POST
	case "PUT":
		return PUT
	case "PATCH":
		return PATCH
	case "DELETE":
		return DELETE
	default:
		return GET
	}
}

const (
	GET    = method("GET")
	POST   = method("POST")
	PUT    = method("PUT")
	PATCH  = method("PATCH")
	DELETE = method("DELETE")
)

type MountPath string

type EndPath struct {
	Method method
	Path   string
}

type AppLink struct {
	Method method
	Path   string
}

type Authorizer interface {
	Authorize(path EndPath, userID string) bool
}

type AuthorizeFunc func(path EndPath, userID string) bool

func (a AuthorizeFunc) Authorize(path EndPath, userID string) bool {
	return a(path, userID)
}

// User is a reduced view of a user
// accustomed to a request/path.
// i.e. the Authorizer is the Authorizer for the app of the current request
// Any further user data that is needed within an app (or parts of an app)
// must be stored inside a new type that is wrapping user
// methods will then receive this wrapped type if needed, so that we have
// type safety. if there are different views needed within different paths
// then we make different wrapper types for them XXXUser etc.
type User struct {
	ID         string
	EMail      string
	Alias      string
	Name       string
	Authorizer Authorizer
}

// Authorize authorizes for the app of the current request
func (u *User) Authorize(path EndPath) bool {
	if u == nil {
		// must not happen, would be an error in composing the app
		panic("user not set")
	}

	if u.ID == "" {
		// must not happen, would be an error in composing the app
		panic("ID not set")
	}

	// must not happen, would be an error in composing the app
	if u.Authorizer == nil {
		panic("Authorizer not set")
	}

	return u.Authorizer.Authorize(path, u.ID)
}

type Authenticator interface {
	Authenticate(r *http.Request) *User
}

type writerTo func(wr io.Writer) (int64, error)

func (w writerTo) WriteTo(wr io.Writer) (int64, error) {
	return w(wr)
}

type ContentType string

func (c ContentType) Mime() string {
	return strings.Split(string(c), ";")[0]
}

type Server interface {
	ContentType() ContentType
	Serve(r *http.Request) (status Status, wt io.WriterTo, err error)
}

type Dispatcher struct {
	get          map[string]Server
	post         map[string]Server
	put          map[string]Server
	del          map[string]Server
	patch        map[string]Server
	NotFound     http.Handler
	ErrorHandler func(r *http.Request, err error)
}

func NewDispatcher() (d *Dispatcher) {
	d = &Dispatcher{}
	d.get = map[string]Server{}
	d.post = map[string]Server{}
	d.put = map[string]Server{}
	d.del = map[string]Server{}
	d.patch = map[string]Server{}
	return
}

func (d *Dispatcher) GETRoutes(mp MountPath) (rt []string) {
	var pre = "/"
	if mp == "" {
		pre = ""
	}
	for r := range d.get {
		rt = append(rt, pre+filepath.Join(string(mp), r))
	}

	return
}

func (d *Dispatcher) POSTRoutes(mp MountPath) (rt []string) {
	var pre = "/"
	if mp == "" {
		pre = ""
	}
	for r := range d.post {
		rt = append(rt, pre+filepath.Join(string(mp), r))
	}

	return
}

func (d *Dispatcher) PUTRoutes(mp MountPath) (rt []string) {
	var pre = "/"
	if mp == "" {
		pre = ""
	}
	for r := range d.put {
		rt = append(rt, pre+filepath.Join(string(mp), r))
	}

	return
}

func (d *Dispatcher) DELETERoutes(mp MountPath) (rt []string) {
	var pre = "/"
	if mp == "" {
		pre = ""
	}
	for r := range d.del {
		rt = append(rt, pre+filepath.Join(string(mp), r))
	}

	return
}

func (d *Dispatcher) PATCHRoutes(mp MountPath) (rt []string) {
	var pre = "/"
	if mp == "" {
		pre = ""
	}
	for r := range d.patch {
		rt = append(rt, pre+filepath.Join(string(mp), r))
	}

	return
}

func (d *Dispatcher) Override(m method, path string, s Server) *Dispatcher {
	if path == "" {
		path = "/"
	}
	switch m {
	case GET:
		d.get[path] = s
	case POST:
		d.post[path] = s
	case PUT:
		d.put[path] = s
	case DELETE:
		d.del[path] = s
	case PATCH:
		d.patch[path] = s
	default:
		d.get[path] = s
	}
	return d
}

func (d *Dispatcher) JSONGet(path string, s JSONServer) *Dispatcher {
	return d.Add(GET, path, s)
}

func (d *Dispatcher) JSONPut(path string, s JSONServer) *Dispatcher {
	return d.Add(PUT, path, s)
}

func (d *Dispatcher) JSONPost(path string, s JSONServer) *Dispatcher {
	return d.Add(POST, path, s)
}

func (d *Dispatcher) JSONDelete(path string, s JSONServer) *Dispatcher {
	return d.Add(DELETE, path, s)
}

func (d *Dispatcher) JSONPatch(path string, s JSONServer) *Dispatcher {
	return d.Add(PATCH, path, s)
}

func (d *Dispatcher) HTMLGet(path string, s HTMLServer) *Dispatcher {
	return d.Add(GET, path, s)
}

func (d *Dispatcher) HTMLPost(path string, s HTMLServer) *Dispatcher {
	return d.Add(POST, path, s)
}

func (d *Dispatcher) Add(m method, path string, s Server) *Dispatcher {
	if path == "" {
		path = "/"
	}
	switch m {
	case GET:
		if s2, has := d.get[path]; has {
			panic(fmt.Sprintf("can't add server %T to GET %#v, path is reserved for %T", s, path, s2))
		}
		d.get[path] = s
	case POST:
		if s2, has := d.post[path]; has {
			panic(fmt.Sprintf("can't add server %T to POST %#v, path is reserved for %T", s, path, s2))
		}
		d.post[path] = s
	case PUT:
		if s2, has := d.put[path]; has {
			panic(fmt.Sprintf("can't add server %T to PUT %#v, path is reserved for %T", s, path, s2))
		}
		d.put[path] = s
	case DELETE:
		if s2, has := d.del[path]; has {
			panic(fmt.Sprintf("can't add server %T to DELETE %#v, path is reserved for %T", s, path, s2))
		}
		d.del[path] = s
	case PATCH:
		if s2, has := d.patch[path]; has {
			panic(fmt.Sprintf("can't add server %T to PATCH %#v, path is reserved for %T", s, path, s2))
		}
		d.patch[path] = s
	default:
		if s2, has := d.get[path]; has {
			panic(fmt.Sprintf("can't add server %T to GET %#v, path is reserved for %T", s, path, s2))
		}
		d.get[path] = s
	}

	return d
}

// Accept: text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8)

func isContentAccepted(c ContentType, r *http.Request) bool {
	ctype := c.Mime()
	hd := r.Header.Get("Accept")
	if hd == "" {
		return true
	}

	// fmt.Printf("accepts: %#v\n", hd)
	acc := strings.Split(hd, ",")

	for _, a := range acc {
		x := strings.Split(strings.TrimSpace(a), ";")

		if len(x) > 0 {
			if x[0] == "*/*" || x[0] == ctype {
				return true
			}
		}
	}

	return false
}

/*
TODO
Accept-Language    de,en-US;q=0.7,en;q=0.3
Accept-Encoding    gzip, deflate
*/

func (d *Dispatcher) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	var m map[string]Server
	switch Method(r) {
	case GET:
		m = d.get
	case POST:
		m = d.post
	case PUT:
		m = d.put
	case DELETE:
		m = d.del
	case PATCH:
		m = d.patch
	default:
		m = d.get
	}

	if s, has := m[r.URL.Path]; has {
		if !isContentAccepted(s.ContentType(), r) {
			wr.WriteHeader(http.StatusExpectationFailed)
			wr.Write([]byte(fmt.Sprintf("content-type %#v not accepted by client", s.ContentType().Mime())))
			return
		}

		status, writeTo, err := s.Serve(r)
		if err != nil && d.ErrorHandler != nil {
			d.ErrorHandler(r, err)
		}
		wr.Header().Set("Content-Type", string(s.ContentType()))
		if status != 0 {
			wr.WriteHeader(int(status))
		}
		_, err = writeTo.WriteTo(wr)
		if err != nil && d.ErrorHandler != nil {
			d.ErrorHandler(r, err)
		}
		return
	}

	if d.NotFound != nil {
		d.NotFound.ServeHTTP(wr, r)
		return
	}

	http.NotFound(wr, r)
}

type JSONServer func(r *http.Request) (status Status, data interface{}, err error)

func (j JSONServer) Serve(r *http.Request) (s Status, wt io.WriterTo, err error) {
	var data interface{}
	s, data, err = j(r)
	if data == nil {
		wt = writerTo(func(wr io.Writer) (i int64, e error) { return })
		return
	}

	wt = writerTo(func(wr io.Writer) (int64, error) {
		return -1, json.NewEncoder(wr).Encode(data)
	})
	return
}

func (j JSONServer) ContentType() ContentType {
	return ContentType("application/json; charset=utf-8")
}

type HTMLer interface {
	HTML() string
}

type HTMLServer func(r *http.Request) (s Status, html HTMLer, err error)

func (h HTMLServer) Serve(r *http.Request) (s Status, w io.WriterTo, err error) {
	var ht HTMLer
	s, ht, err = h(r)

	if ht == nil {
		w = writerTo(func(wr io.Writer) (i int64, e error) {
			_, e = wr.Write([]byte("Page not found"))
			return
		})
		return
	}

	if h, ok := ht.(io.WriterTo); ok {
		w = writerTo(func(wr io.Writer) (i int64, e error) {
			_, e = h.WriteTo(wr)
			return
		})
		return
	}

	data := ht.HTML()

	if data == "" {
		w = writerTo(func(wr io.Writer) (i int64, e error) {
			_, e = wr.Write([]byte("Page not found"))
			return
		})
		return
	}

	w = writerTo(func(wr io.Writer) (i int64, e error) {
		_, e = wr.Write([]byte(string(data)))
		return
	})
	return
}

func (h HTMLServer) ContentType() ContentType {
	return ContentType("text/html; charset=utf-8")
}

// each app handles the internal path dispatching
// the real apps would be wrappers around this more general kind of apps

// each app must have methods that return certain links to its inner workings that may be used inside other apps (AppLink)
// this links must contain the mountpath as prefix
// each app has one global instance that is always used
// and registered to the main app.
// this global instance is used by the other apps to get the AppLinks

// only make mountpath dynamic if really needed (e.g. used in a different webapp)

type App interface {
	http.Handler
	MountPath() MountPath
}

type Main struct {
	mountPaths map[MountPath]App
	NotFound   http.Handler
}

func New() *Main {
	m := &Main{}
	m.mountPaths = map[MountPath]App{}
	return m
}

func (m *Main) Override(apps ...App) {
	for _, a := range apps {
		path := a.MountPath()
		m.mountPaths[path] = a
	}
}

func (m *Main) Add(apps ...App) *Main {
	for _, a := range apps {
		path := a.MountPath()
		if a2, has := m.mountPaths[path]; has {
			panic(fmt.Sprintf("can't register app %T at mountpoint %#v: this mountpoint has already been reserved for app %T", a, path, a2))
		}
		m.mountPaths[path] = a
	}
	return m
}

const fallbackPath = MountPath("")

// gets the app and sets the request path to relative path
// if needed
func (m *Main) getApp(r *http.Request) App {
	idx := strings.Index(r.URL.Path[1:], "/")
	var path = fallbackPath
	if idx == -1 {
		// fmt.Printf("fallback to default: %T\n", m.mountPaths[path])
		return m.mountPaths[path]
	}

	path = MountPath(r.URL.Path[1 : idx+1])

	// fmt.Printf("mountpath: %#v\n", path)

	if a, has := m.mountPaths[path]; has {
		r.URL.Path = r.URL.Path[idx+1:] // set the relative path!
		return a
	}

	return m.mountPaths[fallbackPath]
}

func (m *Main) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	app := m.getApp(r)
	// fmt.Printf("got app %T\n", app)
	if app == nil {
		if m.NotFound != nil {
			m.NotFound.ServeHTTP(rw, r)
			return
		}

		http.NotFound(rw, r)
		return
	}

	// r.Path has a relative path here if needed
	app.ServeHTTP(rw, r)
}
