package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	config "github.com/go-kratos/gateway/api/gateway/config/v1"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/p2c"
)

type BuildContext struct {
	TLSConfigs     map[string]*tls.Config // TLS配置信息
	TLSClientStore *HTTPSClientStore      // HTTPS客户端
}

// Factory is returns service client.
type Factory func(*BuildContext, *config.Endpoint) (Client, error) // 用于返回一个客户端（到服务维度的客户端--会有多个节点）

// *********************************************************************************************************************
type Option func(*options)

type options struct {
	pickerBuilder selector.Builder
}

func WithPickerBuilder(in selector.Builder) Option {
	return func(o *options) {
		o.pickerBuilder = in
	}
}

// *********************************************************************************************************************
func EmptyBuildContext() *BuildContext { // 默认
	return &BuildContext{}
}

// *********************************************************************************************************************
func NewBuildContext(cfg *config.Gateway) *BuildContext { // 参数为配置对象

	tlsConfigs := make(map[string]*tls.Config, len(cfg.TlsStore))

	for k, v := range cfg.TlsStore { // map[string]*TLS

		cfg := &tls.Config{
			InsecureSkipVerify: v.Insecure,
			ServerName:         v.ServerName,
		}

		cert, err := tls.X509KeyPair([]byte(v.Cert), []byte(v.Key))

		if err != nil {

			LOG.Warnf("failed to load tls cert: %q: %v", k, err)

			continue

		}

		cfg.Certificates = []tls.Certificate{cert}

		if v.Cacert != "" {

			roots := x509.NewCertPool()

			if ok := roots.AppendCertsFromPEM([]byte(v.Cacert)); !ok {

				LOG.Warnf("failed to load tls cacert: %q", k)

				continue

			}

			cfg.RootCAs = roots

		}

		tlsConfigs[k] = cfg

	}

	return &BuildContext{
		TLSConfigs:     tlsConfigs,
		TLSClientStore: NewHTTPSClientStore(tlsConfigs),
	}

}

// NewFactory new a client factory.
func NewFactory(r registry.Discovery, opts ...Option) Factory {

	o := &options{
		pickerBuilder: p2c.NewBuilder(), // 默认选择器BUILDER -- 可以自己指定
	}

	for _, opt := range opts {
		opt(o)
	}

	return func(builderCtx *BuildContext, endpoint *config.Endpoint) (Client, error) { // 客户端工厂：用于实时获取一个客户端

		picker := o.pickerBuilder.Build() // selector.Selector

		ctx, cancel := context.WithCancel(context.Background())

		applier := &nodeApplier{
			cancel:       cancel,
			endpoint:     endpoint,
			registry:     r,
			picker:       picker,
			buildContext: builderCtx,
		}

		if err := applier.apply(ctx); err != nil {
			return nil, err
		}

		client := newClient(applier, picker)

		return client, nil

	}

}

type nodeApplier struct {
	canceled     int64              // ???
	buildContext *BuildContext      // TLS相关
	cancel       context.CancelFunc // CONTEXT的取消函数
	endpoint     *config.Endpoint   // 配置信息
	registry     registry.Discovery // 服务发现
	picker       selector.Selector  // 选择器（所有发现的服务节点都会应用到PICKER中）
}

func (na *nodeApplier) apply(ctx context.Context) error {

	var nodes []selector.Node

	for _, backend := range na.endpoint.Backends { // 指定的后端地址

		target, err := parseTarget(backend.Target)

		if err != nil {
			return err
		}

		switch target.Scheme {
		case "direct":

			weighted := backend.Weight // weight is only valid for direct scheme

			node := newNode(na.buildContext, backend.Target, na.endpoint.Protocol, weighted, backend.Metadata, "", "", WithTLS(backend.Tls), WithTLSConfigName(backend.TlsConfigName))

			nodes = append(nodes, node)

			na.picker.Apply(nodes)

		case "discovery":

			existed := AddWatch(ctx, na.registry, target.Endpoint, na)

			if existed {
				log.Infof("watch target %+v already existed", target)
			}

		default:

			return fmt.Errorf("unknown scheme: %s", target.Scheme)

		}

	}

	return nil

}

var _defaultWeight = int64(10)

func nodeWeight(n *registry.ServiceInstance) *int64 {

	w, ok := n.Metadata["weight"]

	if ok {

		val, _ := strconv.ParseInt(w, 10, 64)

		if val <= 0 {
			return &_defaultWeight
		}

		return &val

	}

	return &_defaultWeight

}

func (na *nodeApplier) Callback(services []*registry.ServiceInstance) error { // 回调函数：当服务发现发生变化时会调用这个函数

	if atomic.LoadInt64(&na.canceled) == 1 { // 取消了
		return ErrCancelWatch
	}

	if len(services) == 0 {
		return nil
	}

	scheme := strings.ToLower(na.endpoint.Protocol.String())

	nodes := make([]selector.Node, 0, len(services))

	for _, ser := range services {

		addr, err := parseEndpoint(ser.Endpoints, scheme, false)

		if err != nil || addr == "" {

			log.Errorf("failed to parse endpoint: %v/%s: %v", ser.Endpoints, scheme, err)

			continue

		}

		node := newNode(na.buildContext, addr, na.endpoint.Protocol, nodeWeight(ser), ser.Metadata, ser.Version, ser.Name, WithTLS(false))

		nodes = append(nodes, node)

	}

	na.picker.Apply(nodes)

	return nil

}

func (na *nodeApplier) Cancel() {

	log.Infof("Closing node applier for endpoint: %+v", na.endpoint)

	atomic.StoreInt64(&na.canceled, 1)

	na.cancel()

}

func (na *nodeApplier) Canceled() bool {
	return atomic.LoadInt64(&na.canceled) == 1
}
