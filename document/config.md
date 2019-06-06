### [配置文件属性介绍](./config.toml)
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

#### 8. [[kafka]]  kafka配置
> 所有的topic必须是key,val形式,且key为websocket订阅所用
```
{
    enable:"bool",//是否启用
    topics:[{
            brokers:"[]string",
            topic:"string",
            offset:"int64", // log偏移量 
            groupId:"string",
            partition:"int", 
            minBytes:"int",
            maxBytes:"int", 
            maxWait:"timeStr", 
        }
    ]
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