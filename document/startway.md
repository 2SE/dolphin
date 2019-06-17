### 单节点启动
```
[cluster]
self = ""
```

> 设置self为空时，为单节点启动

### cluster 启动
 对于集群，dolphin使用了raft共识算法，最小集群单位为3个节点。
 以下提供了基础3节点中cluster部分配置方案。（如果用于测试，可以配置单击，区分端口号即可，不过对于dolphin而言除了集群配置还有其他如ws端口等，所以配置的时候需要注意端口的管理）

##### node1
```
[cluster]
self = "node1"

[[cluster.nodes]]
name = "node1"
address = "ip1:port"
[[cluster.nodes]]
name = "node2"
address = "ip2:port"
[[cluster.nodes]]
name = "node3"
address = "ip3:port"

[cluster.connection]
dial_timeout = "5s"
max_delay = "1m"
base_delay = "1s"
factor = 1.6
jitter = 0.2
disable_request_timeout = false
request_wait_after = "1m"
[cluster.failover]
enabled = true
heartbeat = "100ms"
vote_after = 8
node_fail_after = 16
```
##### node2
```
[cluster]
self = "node2"

[[cluster.nodes]]
name = "node1"
address = "ip1:port"
[[cluster.nodes]]
name = "node2"
address = "ip2:port"
[[cluster.nodes]]
name = "node3"
address = "ip3:port"

[cluster.connection]
dial_timeout = "5s"
max_delay = "1m"
base_delay = "1s"
factor = 1.6
jitter = 0.2
disable_request_timeout = false
request_wait_after = "1m"
[cluster.failover]
enabled = true
heartbeat = "100ms"
vote_after = 8
node_fail_after = 16
```

##### node3
```
[cluster]
self = "node3"

[[cluster.nodes]]
name = "node1"
address = "ip1:port"
[[cluster.nodes]]
name = "node2"
address = "ip2:port"
[[cluster.nodes]]
name = "node3"
address = "ip3:port"

[cluster.connection]
dial_timeout = "5s"
max_delay = "1m"
base_delay = "1s"
factor = 1.6
jitter = 0.2
disable_request_timeout = false
request_wait_after = "1m"
[cluster.failover]
enabled = true
heartbeat = "100ms"
vote_after = 8
node_fail_after = 16
```