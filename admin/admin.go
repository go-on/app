package admin

import (
	"github.com/go-on/app"
	"net/http"
)

// Admin is the app that always needs a user
// so it tries to get the user from the request via
// Authenticator
// the Login for the Admin is an extra app that sets the
// user-id somewhere so that the Authenticator may find it
type Admin struct {
	user *app.User
	app.Authenticator
	app.Authorizer
	*app.Dispatcher
}

func (a *Admin) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	if a.Authenticator == nil {
		panic("no authenticator set")
	}
	if a.Authorizer == nil {
		panic("no authorizer set")
	}

	user := a.Authenticator.Authenticate(r)
	if user == nil {
		rw.WriteHeader(http.StatusForbidden)
		return
	}
	a.user = user
	a.user.Authorizer = a.Authorizer
	path := app.EndPath{app.Method(r), r.URL.Path}
	if !a.user.Authorize(path) {
		rw.WriteHeader(http.StatusForbidden)
		return
	}
	a.Dispatcher.ServeHTTP(rw, r)
}

// each app handles the internal path dispatching
// the real apps would be wrappers around this more general kind of apps

// each app must have methods that return certain links to its inner workings that may be used inside other apps (AppLink)
// this links must contain the mountpath as prefix
// each app has one global instance that is always used
// and registered to the main app.
// this global instance is used by the other apps to get the AppLinks

// only make mountpath dynamic if really needed (e.g. used in a different webapp)
func (a Admin) MountPath() app.MountPath {
	return app.MountPath("admin")
}

var _ app.App = &Admin{}
