package login

import (
	"github.com/go-on/app"
	"net/http"
)

type Login struct {
	*app.Dispatcher
}

func (l Login) MountPath() app.MountPath {
	return app.MountPath("login")
}

func (l *Login) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	l.Dispatcher.ServeHTTP(rw, r)
}

var _ app.App = &Login{}
