# GATEWAY

入口文件：cmd/gateway/main.go

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


工作流程：
1. 服务初始流程
> 解析命令行参数->开启PPROF服务->读取解析配置文件，同时监控该配置文件->通过配置文件建立代理模型->KRATOS APP启动服务
2. 请求流程
> 请求进入->公共中间件->单服务配置中间件->对请求进行处理->发送请求->接收响应再次进入中间件->输出响应结果



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