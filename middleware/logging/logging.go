package logging

import (
	"net/http"
	"strings"
	"time"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"
	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/log"
)

func init() {
	middleware.Register("logging", Middleware)
}

// Middleware is a logging middleware.
func Middleware(c *config.Middleware) (middleware.Middleware, error) { // 将配置转换成中间件对象

	return func(next http.RoundTripper) http.RoundTripper { // middleware.Middleware：func(http.RoundTripper) http.RoundTripper

		// RoundTripperFunc：func(*http.Request) (*http.Response, error)
		// RoundTripperFunc：具有RoundTrip方法
		// http.RoundTripper：RoundTrip(*Request) (*Response, error)
		return middleware.RoundTripperFunc(func(req *http.Request) (reply *http.Response, err error) { // 当前函数被转换为RoundTripperFunc，并具有RoundTrip方法

			startTime := time.Now()

			reply, err = next.RoundTrip(req)

			level := log.LevelInfo

			code := http.StatusBadGateway // 默认状态码（BadGateway）

			errMsg := ""

			if err != nil {

				level = log.LevelError

				errMsg = err.Error()

			} else {
				code = reply.StatusCode
			}

			ctx := req.Context()

			// nodes, _ := middleware.RequestBackendsFromContext(ctx)

			reqOpt, _ := middleware.FromRequestContext(ctx)

			log.Context(ctx).Log(level,
				"source", "accesslog",
				"host", req.Host,
				"method", req.Method,
				"scheme", req.URL.Scheme,
				"path", req.URL.Path,
				"query", req.URL.RawQuery,
				"code", code,
				"error", errMsg,
				"latency", time.Since(startTime).Seconds(),
				"backend", strings.Join(reqOpt.Backends, ","),
				"backend_code", reqOpt.UpstreamStatusCode,
				"backend_latency", reqOpt.UpstreamResponseTime,
				"last_attempt", reqOpt.LastAttempt,
			)

			return reply, err

		})

	}, nil

}
