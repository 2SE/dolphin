![dolphin logo](./document/256px%20png.png)

## 什么是dolphin
dolphin是一个集成了api网关，服务发现，请求限流，一致性hash路由，服务调度的统一接入框架，旨在对微服务的开发和依赖做到更加智能的管理，加快开发速度，解决中小团队开发微服务的难度。
- 对于数据的传输使用protobuf，能够更高效率的完成数据传输。
- 对于相同的服务可以集群部署，分流服务压力，dolphin本身也支持集群，服务注册需要指定dolphin节点。
- 提供一致性hash进行请求分流，能够满足相同服务增加或者减少时的请求的均匀分布，疏解服务器压力。
- 提供后端服务之间的依赖调度，使服务依赖管理更简单。
- 提供对外websocket请求限流，对内grpc服务路由没做限制。
- 提供白名单接口放行设置，可以有效的管理对外接口的基本限行。
- 提供kafka订阅机制，完成前端订阅自动分发。

## dolphin的概念
当一个团队着手微服务的开发时，往往面对很多抉择，因为微服务存在很多缺点，如系统间api互相访问时，服务的高度依赖耦合难以维护；系统负载增加时，难以进行水平扩展，要面对各式各样的工具组合，增加运维难度。dolphin的出现将这些痛点化繁为简，让项目的开发部署更为方便。



## go依赖
1. Go 1.11 or newer.
2. go mod 
## 如何使用

### 1. protobuf全局请求体
- [dolphin/pb/appserve.proto](./pb/appserve.proto)用于标准化请求的参数和请求的返回参数

- [dolphin/pb/login.proto](./pb/login.proto)用户约束登录成功后对用户id的监控，并使用截获的用户id做一致性hash

### 2. 运行流程
![single dolphin](./document/single%20dolphin.png)
 
- dolphin启动后首先将各个服务通过dolphin的http端口发送服务注册请求，dolphin在接收到请求后会将服务连接持久化并执行健康监测，如果注册的多个服务是相同的，dolphin内部会在调用时通过用户一致性哈希路由，服务之间的调用则通过dolphin内部的grpc-server解耦

- websocket支持订阅，通过约定好的订阅key，服务在运行时可以将用户订阅的信息直接以kv的形式写入到kafka，dolphin会消费指定topic的内容分发给订阅的通道。

### 启动dolphin
> [配置文件参数介绍](./document/config.md)
#### 1. 单节点启动流程
1. 下载dolphin源码
>git clone https://github.com/2SE/dolphin.git
2. 获取代码依赖
>go mod download
3. 编译dolphin
>cd ./cmd/dolphin \
go build -o dolphin
4. 将[配置文件](./document/simple.toml)复制到dolphin运行文件同级目录并启动
>./dolphin -c simple.toml

#### 2. 集群启动流程
首先对照[startway.md](./document/startway.md)中对于单点和集群的差异修改好配置文件，然后以单点启动的方式逐个启动dolphin节点即可。

### 微服务依赖dolphin掉用示例（支持单节及多节点）
#### 1. 启动测试用服务端注册到dolphin
```
在dolphin/cmd/serverexample目录下有一个测试用的Grpc服务 
在server.go 中有 
    dolphinAddr = "http://dolphin_ip:dolpin_port"
    //如果与dolphin不在同一台机器请使用内网/外网ip
    appInfo.Address = "local_ip:grpc_server_port" 
修改这两项适配自身环境后运行"go run",
             	
```

#### 2. 启动测试客户端发起请求
```
在dolphin/cmd/performance目录下是一个ws请求示例，配合serverexample使用，
该测试用例可结合dolphin中pprof测试dolphin性能。具体pprof使用请自行参考网
上案例。

var (
	//dolphin websocket地址
	addr = "dolphin_ip:ws_port"
)

func main() {
	conns := GetClients(1) //设置生成客户端数量
	for _, v := range conns {
		go func(conn *websocket.Conn) {
			req := getRequests(1) //设置单个客户端串行请求次数
			sendRequest(conn, req)
		}(v)
	}
	time.Sleep(time.Second * 5)
}

修改完配置后运行"go run即可"
```