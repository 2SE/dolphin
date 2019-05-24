![dolphin logo](./256px%20png.png)

## What is dolphin
dolphin是一个集成了api网关，服务发现，请求限流，一致性hash路由，服务调度的统一接入框架，旨在对微服务的开发和依赖做到更加智能的管理，加快开发速度，解决中小团队开发微服务的难度。
- 对于数据的传输使用protobuf，能够更高效率的完成数据传输。
- 对于相同的服务可以集群部署，分流服务压力，dolphin本身也支持集群，服务注册需要指定dolphin节点。
- 提供一致性hash进行请求分流，能够满足相同服务增加或者减少时的请求的均匀分布，疏解服务器压力。
- 提供后端服务之间的依赖调度，使服务依赖管理更简单。
- 提供对外websocket请求限流，对内grpc服务路由没做限制。
- 提供白名单接口放行设置，可以有效的管理对外接口的基本限行。
- 提供kafka订阅机制，完成前端订阅自动分发。

## Design concepts
当一个团队着手微服务的开发时，往往面对很多抉择，因为微服务存在很多缺点，如系统间api互相访问时，服务的高度依赖耦合难以维护；系统负载增加时，难以进行水平扩展，要面对各式各样的工具组合，增加运维难度。dolphin的出现将这些痛点化繁为简，让项目的开发部署更为方便。



## Requirements
Go 1.11 or newer.

## How to use

### 1. protobuf全局请求体
- [dolphin/pb/appserve.proto](./pb/appserve.proto)用于标准化请求的参数和请求的返回参数

- [dolphin/pb/login.proto](./pb/login.proto)用户约束登录成功后对用户id的监控，并使用截获的用户id做一致性hash

### 2. 运行流程
![single dolphin](./single%20dolphin.png)
 
- dolphin启动后首先将各个服务通过dolphin的http端口发送服务注册请求，dolphin在接收到请求后会将服务连接持久化并执行健康监测，如果注册的多个服务是相同的，dolphin内部会在调用时通过用户一致性哈希路由，服务之间的调用则通过dolphin内部的grpc-server解耦

- websocket支持订阅，通过约定好的订阅key，服务在运行时可以将用户订阅的信息直接以kv的形式写入到kafka，dolphin会消费指定topic的内容分发给订阅的通道。
### 3. [配置文件](./config.toml)
#### 1. [websocket] 对外提供访问的配置
```
{
    listen:"string", //监听端口号
    read_buf_size:"int", //读取缓冲区大小设置
    write_buf_size:"int", //写缓冲区大小设置
    write_time_wait:"timeStr", //写数据超时,ps:"2s"
    read_time_wait:"timeStr", // 读数据超时
    tls:{
        enabled:"bool", //是否启用
        http_redirect:"string",//请求重定向
        cert_file:"string",//cert 路径
        key_file:"string",// key 路径
        autocert:{
            cert_cache:"string",
            email:"string",
            domains:"[]string",
        }
    } 
    idle_session_timeout:"timeStr", // 空闲会话超时
    session_queue_size:"int", // 写数据个数缓冲区限制
    queue_out_timeout:"timeStr", // 写数据个数缓冲区消费等待时间
    id_salt:"string", // sess.guid 盐
}

```

#### 2. [routehttp]  grpc服务注册的配置
```
address grpc服务注册端口
```

#### 3. [scheduler] grpc对内提供服务间相互调用的配置
```
address grpc内部服务调用端口
```
#### 4. [route] 调用路径配置
```
{
    heartBeat:"timeStr", //dolphin维护的grpc client的心跳监测周期
    recycle:"timeStr", // grpc client 回收周期
    threshold:"int16", //grpc client 不正常请求个数阙值，超过个数后，在下次回收周期到来后grpc client会被移除
    timeout:"timeStr", // grpc client 请求超时
}
```
#### 5. [limit]  请求限流配置
```
{
    maxNum:"int", // 最大请求数
    maxRate:"int", // 每个bucket每秒钟请求个数
    maxBurst:"int", // 每个bucket可溢出请求个数
}
```

#### 6. [login] 登录接口监测设置
```
{
    resource:"string", // 资源路径
    version:"string", // 版本号
    action:"string", // 方法名
}
```

#### 7. [[whiteList]]  放行方法设置
```
{
    resource:"string", // 资源路径
    version:"string", // 版本号
    action:"string", // 方法名
}
```

#### 8. [[kafkas]]  kafka配置
> 所有的topic必须是key,val形式,且key为websocket订阅所用
```
{
    brokers:"[]string",
    topic:"string",
    offset:"int64", // log偏移量 
    groupId:"string",
    partition:"int", 
    minBytes:"int",
    maxBytes:"int", 
    maxWait:"timeStr", 
}
```

#### 9. [cluster] 集群配置
```
{
    self:"string", //节点名称
    connection: {
        dial_timeout:"time", //拨号超时时间
        max_delay:"time", // 重试时间上限
        base_delay:"time", // 第一次重试等待时间
        factor:"float64", // 重试因素
        jitter:"float64", //抖动因子
        disable_request_timeout:"bool", //禁用请求超时
        request_wait_after:"time",//请求超时
    }
    nodes:[
        {
            name:"string", //节点名称
            address:"string",//节点路径
        }
    ]
    failover:{
        enabled:"bool", // 健康检查是否启用
        heartbeat:"time", //心跳周期
        vote_after:"int", // 投票暂停:在新选举开始前心跳停止的次数
        node_fail_after:"int", //一个节点被宣布死亡的心跳的次数
    }
}
```