package debug

import (
	"net/http"
	"net/http/pprof"
	"path"
	"strings"

	rmux "github.com/go-kratos/gateway/router/mux"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/mux"
)

const (
	_debugPrefix = "/debug"
)

// *********************************************************************************************************************
var globalService = &debugService{
	handlers: map[string]http.HandlerFunc{ // 优先处理
		"/debug/ping":               func(rw http.ResponseWriter, r *http.Request) {},
		"/debug/pprof/":             pprof.Index,
		"/debug/pprof/cmdline":      pprof.Cmdline,
		"/debug/pprof/profile":      pprof.Profile,
		"/debug/pprof/symbol":       pprof.Symbol,
		"/debug/pprof/trace":        pprof.Trace,
		"/debug/pprof/allocs":       pprof.Handler("allocs").ServeHTTP,
		"/debug/pprof/block":        pprof.Handler("block").ServeHTTP,
		"/debug/pprof/goroutine":    pprof.Handler("goroutine").ServeHTTP,
		"/debug/pprof/heap":         pprof.Handler("heap").ServeHTTP,
		"/debug/pprof/mutex":        pprof.Handler("mutex").ServeHTTP,
		"/debug/pprof/threadcreate": pprof.Handler("threadcreate").ServeHTTP,
	},
	mux: mux.NewRouter(), // 兜底（接受注册）
}

func Register(name string, debuggable Debuggable) {
	globalService.Register(name, debuggable)
}

func MashupWithDebugHandler(origin http.Handler) http.Handler { // 将原始HANDLER和全局DEBUG相关的HANDLER合并

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		if strings.HasPrefix(req.URL.Path, _debugPrefix) { // 表示以DEBUG前缀开头的路径

			rmux.ProtectedHandler(globalService).ServeHTTP(w, req) // X-Forwarded-For：如果有则拒绝

			return

		}

		origin.ServeHTTP(w, req)

	})

}

// *********************************************************************************************************************
type Debuggable interface {
	DebugHandler() http.Handler // 返回HANDLER用于处理指定路径
}

// *********************************************************************************************************************
type debugService struct {
	handlers map[string]http.HandlerFunc
	mux      *mux.Router // 兜底的HANDLER：http.Handler
}

func (d *debugService) ServeHTTP(w http.ResponseWriter, req *http.Request) { // 接口：http.Handler

	for path, handler := range d.handlers { // 先处理

		if path == req.URL.Path {

			handler(w, req)

			return

		}

	}

	d.mux.ServeHTTP(w, req) // 兜底的

}

func (d *debugService) Register(name string, debuggable Debuggable) {

	path := path.Join(_debugPrefix, name)

	d.mux.PathPrefix(path).Handler(debuggable.DebugHandler())

	log.Infof("register debug: %s", path)

}

// *********************************************************************************************************************
