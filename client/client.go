package client

import (
	"io"
	"net/http"
	"time"

	"github.com/go-kratos/gateway/middleware"
	"github.com/go-kratos/kratos/v2/selector"
)

type client struct { // 到后端服务端的客户端
	applier  *nodeApplier      // ？？？干啥用
	selector selector.Selector // ？？？
}

type Client interface { // 接口
	http.RoundTripper
	io.Closer
}

func newClient(applier *nodeApplier, selector selector.Selector) *client {
	return &client{
		applier:  applier,
		selector: selector,
	}
}

func (c *client) Close() error {

	c.applier.Cancel()

	return nil

}

func (c *client) RoundTrip(req *http.Request) (resp *http.Response, err error) { // http.RoundTripper

	ctx := req.Context()

	reqOpt, _ := middleware.FromRequestContext(ctx) // ？？？在哪存入

	filter, _ := middleware.SelectorFiltersFromContext(ctx) // 隶属于上面的对象：RequestOptions

	n, done, err := c.selector.Select(ctx, selector.WithNodeFilter(filter...)) // 执行节点选择

	if err != nil {
		return nil, err
	}

	reqOpt.CurrentNode = n

	addr := n.Address()

	reqOpt.Backends = append(reqOpt.Backends, addr)

	// *********************************************************
	backendNode := n.(*node)

	// 这里是直接调整原始请求的参数
	req.URL.Host = addr
	req.URL.Scheme = "http"

	if backendNode.tls {
		req.URL.Scheme = "https"
		req.Host = addr
	}

	if nodeHost := n.Metadata()["host"]; nodeHost != "" {
		req.Host = nodeHost
	}

	req.RequestURI = "" // TODO 必须置空

	startAt := time.Now()

	resp, err = backendNode.client.Do(req) // TODO 真正的执行请求

	reqOpt.UpstreamResponseTime = append(reqOpt.UpstreamResponseTime, time.Since(startAt).Seconds())

	if err != nil {

		done(ctx, selector.DoneInfo{Err: err})

		reqOpt.UpstreamStatusCode = append(reqOpt.UpstreamStatusCode, 0)

		return nil, err

	}

	reqOpt.UpstreamStatusCode = append(reqOpt.UpstreamStatusCode, resp.StatusCode)
	reqOpt.DoneFunc = done

	return resp, nil

}
