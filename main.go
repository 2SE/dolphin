package main

import (
	"github.com/2se/dolphin/cluster"
	"github.com/2se/dolphin/config"
	"log"
)

func main() {
	cnf, err := config.Load("")
	if err != nil {
		log.Fatalf("载入配置文件失败： %v", err)
		return
	}

	if cnf == nil {
		log.Printf("配置文件为空")
		return
	}

	log.Printf("配置文件信息: %s", cnf)

	workerId, err := cluster.Init(cnf.GetClusterConfig())
	if err != nil {
		log.Printf("初始化集群失败 %v", err)
		return
	}

	log.Printf("当前节点运行的workid %d", workerId)

	cluster.Start()
	defer cluster.Shutdown()

	cluster.Route()
}
