package public

import (
	"github.com/go-on/app"
	"net/http"
)

type Public struct {
	*app.Dispatcher
}

func (p *Public) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	p.Dispatcher.ServeHTTP(rw, r)
}

// the empty mountpath is the fallback for every path
func (p Public) MountPath() app.MountPath {
	return app.MountPath("")
}

var _ app.App = &Public{}

func New() *Public {
	return &Public{app.NewDispatcher()}
}
