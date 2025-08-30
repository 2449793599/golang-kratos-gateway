# GATEWAY

API网关：基于KRATOS微服务框架的API网关（算是KRATOS的一个简要使用案例）

入口文件：cmd/gateway/main.go

### 1. 项目结构
```
|-cmd/gateway
|    |-main.go          入口总文件，读取配置，启动KRATOS-APP
|-server/server.go      网关服务，注入KKRATOS-APP
|-proxy/proxy.go        核心代理转发模块
|-router                路由模块
|    |-mux/mux.go       MUX路由
|-client                CLIENT模块，本质上是对HTTP.NET.CLIENT的扩展
|    |-factory.go       CLIENT FACTORY，依据CONFIG.ENDPOINT来注册CLIENT
|    |-client.go        CLIENT代理结构体
|    |-node.go          SELECTOR.NODE接口的实现，对访问节点进行选择
|    |-retry.go         重试机制实现
|    |-target.go        解析最终访问目标
|-middleware            中间件模块
|    |-middleware.go                中间件接口
|    |-registry.go                  中间件注册机制实现
|    |-request.go                   中间件请求配置关联上下文
|    |-logging.go                   LOG关联上下文
|    |-color/color.go               对"染色"的HEADER的请求进行过滤
|    |-cors/cors.go                 跨域实现
|    |-logging/logging.go           日志中间件
|    |-otel/otel.go                 OPENTELEMETRY的TRACER对接实现
|    |-prometheus/prometheus.go     PROMETHEUS监控
```
GRPC代理则是修改CLIENT，基本与HTTP代理大同小异，主要的不同点就是使用了HTTP/2。

### 2. 配置文件

配置文件：cmd/gateway/config.yaml

解析对象：api/gateway/config/v1/gateway.proto
解析流程：
```
cfg := config.New(
    config.WithSource(
        file.NewSource("config.yaml"),
    ),
)

if err := cfg.Load(); err != nil {
    t.Fatal(err)
}
```
配置详情参考：config.demo.yaml

工作流程：
1. 服务初始流程

- 解析命令行参数
- 开启PPROF服务
- 读取解析配置文件，同时监控该配置文件
- 通过配置文件建立代理模型
- KRATOS-APP启动服务

2. 请求流程

- 请求进入
- 公共中间件
- 单服务配置中间件
- 对请求进行处理
- 发送请求
- 接收响应再次进入中间件
- 输出响应结果



HTTP -> Proxy -> Router -> Middleware -> Client -> Selector -> Node

## Protocol
* HTTP -> HTTP
* HTTP -> gRPC
* gRPC -> gRPC

## Encoding
* Protobuf Schemas

## Endpoint
* prefix: /api/echo/*
* path: /api/echo/hello
* regex: /api/echo/[a-z]+
* restful: /api/echo/{name}

## Middleware
* cors
* auth
* color
* logging
* tracing
* metrics
* ratelimit
* datacenter