package main

import (
	"fmt"
	"github.com/go-on/app"
	"github.com/go-on/app/admin"
	"github.com/go-on/app/files"
	"github.com/go-on/app/login"
	"github.com/go-on/app/public"
	"github.com/go-on/lib/html"
	"github.com/go-on/lib/types"
	"path/filepath"
	// "github.com/go-on/lib/html/element"
	"net/http"
)

// each app handles the internal path dispatching
// the real apps would be wrappers around this more general kind of apps

// each app must have methods that return certain links to its inner workings that may be used inside other apps (AppLink)
// this links must contain the mountpath as prefix
// each app has one global instance that is always used
// and registered to the main app.
// this global instance is used by the other apps to get the AppLinks

// only make mountpath dynamic if really needed (e.g. used in a different webapp)

type MainApp struct {
	*app.Main
}

func (m *MainApp) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s %#v\n", r.Method, r.URL.Path)
	m.Main.ServeHTTP(wr, r)
}

var (
	filesDir1      = "."
	filesDir2      = filepath.Join(filesDir1, "files")
	Main           = &MainApp{app.New()}
	Admin          = &admin.Admin{}
	Login          = &login.Login{}
	PublicFiles    = files.New("static", filesDir1)
	Public         = newMyPublic()
	PublicApi      = &publicApi{public.New()}
	ProtectedFiles = files.New("protected", filesDir2).Protected(nil, nil)
)

type publicApi struct {
	*public.Public
}

func (a *publicApi) MountPath() app.MountPath {
	return "api"
}

func newMyPublic() *myPublic {
	return &myPublic{
		sidebars: map[string]app.HTMLer{},
		Public:   public.New(),
	}
}

type myPublic struct {
	*public.Public
	sidebars map[string]app.HTMLer
}

func (m *myPublic) AddSidebar(path string, ht app.HTMLer) {
	m.sidebars[path] = ht
}

func (m *myPublic) Navigation(r *http.Request) app.HTMLer {
	ul := html.UL()

	for _, rt := range m.Public.Dispatcher.GETRoutes(m.MountPath()) {
		a := html.AHref(rt, rt)
		if filepath.Join(string(m.MountPath()), r.URL.Path) == rt {
			a.Add(types.Style{"color", "red"})
		}
		ul.Add(html.LI(a))
	}

	return ul
}

func (m *myPublic) ApiNavigation() app.HTMLer {
	ul := html.UL()

	for _, rt := range PublicApi.Dispatcher.GETRoutes(PublicApi.MountPath()) {
		ul.Add(html.LI(html.AHref(rt, rt)))
	}

	return ul
}

func (m *myPublic) SideBar(r *http.Request) app.HTMLer {
	path := r.URL.Path
	if sb, has := m.sidebars[path]; has {
		return sb
	}

	return html.DIV()
}

func (m *myPublic) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	html.HTML5(
		html.BODY(
			html.H1("my site"),
			m.Public,
			html.HR(),
			m.SideBar(r),
			html.H2("Pages"),
			m.Navigation(r),
			html.H2("API"),
			m.ApiNavigation(),
		),
	).ServeHTTP(wr, r)
}

func index(r *http.Request) (s app.Status, ht app.HTMLer, err error) {
	ht = html.H1("index")
	return
}

func jsonHu(r *http.Request) (s app.Status, data interface{}, err error) {
	data = map[string]string{"json": "hu"}
	return
}

func errorTest(r *http.Request) (s app.Status, ht app.HTMLer, err error) {
	ht = html.H1("Testfehler ")

	s = http.StatusNotFound
	err = fmt.Errorf("hier ist ein testfehler")
	return
}

func htmlHu(r *http.Request) (s app.Status, ht app.HTMLer, err error) {
	ht = html.DIV(
		html.H1("html "+r.URL.Path),
		html.AHref(PublicFiles.Link("text.txt").Path, "to text.txt"),
	)
	return
}

func errorHandlerPublic(r *http.Request, err error) {
	fmt.Printf("Error in public app %s %#v: %s\nHeaders: %v\n", r.Method, r.URL.Path, err.Error(), r.Header)
}

func init() {
	Public.Dispatcher.ErrorHandler = errorHandlerPublic
	Public.Dispatcher.
		HTMLGet("/", index).
		HTMLGet("/hu", htmlHu).
		HTMLGet("/hu/ho", htmlHu).
		HTMLGet("/error", errorTest)

	Public.AddSidebar("/hu", html.H3("hu-sidebar"))

	PublicApi.Dispatcher.
		JSONGet("/hu.json", jsonHu)

	Main.Add(Admin, Login, Public, PublicFiles, ProtectedFiles, PublicApi)
}

func main() {
	http.ListenAndServe(":8080", Main)
}
