package cluster

import (
	"bufio"
	"encoding/gob"
	"errors"
	"github.com/2se/dolphin/common/backoff"
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"sync"
	"time"
)

// peer 对于集群模式下与自身对等的dolphin节点，视作peer
type peer struct {
	lock         sync.RWMutex
	endpoint     *rpc.Client
	connected    bool
	reconnecting bool
	address      string
	name         string
	failCount    int
	done         chan bool
}

func (n *peer) disconnect() {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.reconnecting {
		n.done <- true
		return
	}

	if n.connected {
		err := n.endpoint.Close()
		log.WithField("error", err).Infof("disconnect remote peer '%s'", n.name)
	}
}

func (n *peer) reconnect() {
	// Avoid parallel reconnection threads
	n.lock.Lock()
	if n.reconnecting {
		n.lock.Unlock()
		return
	}
	n.reconnecting = true
	n.lock.Unlock()
	var (
		wait        *time.Timer
		count       int
		err         error
		conn        net.Conn
		dialTimeout time.Duration
	)

	if peerConnCnf != nil {
		dialTimeout = peerConnCnf.DialTimeout.Get()
	} else {
		dialTimeout = defaultDialTimeout
	}

	// 重连的间隔时间随着重试次数的增加而延长。最长延迟根据配置设定。
	nextWait := backoff.New(peerConnCnf)
	for {
		log.WithField("dialTimeout", dialTimeout).
			WithField("address:", n.address).
			WithField("name", n.name).
			Debug("cluster: dial to remote peer..")
		conn, err = net.DialTimeout(tcpNetwork, n.address, dialTimeout)
		// Attempt to reconnect right away
		if err == nil {
			n.lock.Lock()
			if n.reconnecting == false {
				conn.Close()
				n.lock.Unlock()
				return
			}
			n.lock.Unlock()

			n.newClient(conn)
			if wait != nil {
				wait.Stop()
			}

			n.lock.Lock()
			n.connected = true
			n.reconnecting = false
			n.lock.Unlock()
			log.Infof("cluster: connection to '%s' established", n.name)
			return
		} else if count == 0 {
			wait = time.NewTimer(defaultClusterReconnect)
		} else {
			wait.Reset(nextWait.Duration(count))
		}

		count++
		if count > 10 {
			//log.Errorf("cluster: reconnect to remote peer failed more than 10. %v", err)
		}

		select {
		case <-wait.C:
			// Wait for timer to try to reconnect again. Do nothing if the timer is inactive.
		case <-n.done:
			// Shutting down
			log.Infof("cluster: peer '%s' shutdown started", n.name)
			wait.Stop()
			if n.endpoint != nil {
				n.endpoint.Close()
			}
			n.lock.Lock()
			n.connected = false
			n.reconnecting = false
			n.lock.Unlock()
			log.Infof("cluster: peer '%s' shut down completed", n.name)
			return
		}
	}
}

func (n *peer) newClient(conn net.Conn) {
	if peerConnCnf != nil && peerConnCnf.DisableReqTimeout {
		n.endpoint = rpc.NewClient(conn)
		return
	}

	encBuf := bufio.NewWriter(conn)
	codec := &gobClientCodec{conn, gob.NewDecoder(conn), gob.NewEncoder(encBuf), encBuf}
	n.endpoint = rpc.NewClientWithCodec(codec)
}

func (n *peer) call(serviceMethod string, req, resp interface{}) error {
	n.lock.RLock()
	if !n.connected {
		n.lock.RUnlock()
		return errors.New("cluster: peer '" + n.name + "' not connected")
	}
	n.lock.RUnlock()
	// 如果请求远端数据失败，将重连远端的连接，本次请求将失败返回
	if err := n.endpoint.Call(serviceMethod, req, resp); err != nil {
		log.Infof("cluster: call failed to '%s' [%s]", n.name, err)
		n.lock.Lock()
		if n.connected {
			n.endpoint.Close()
			n.connected = false
			go n.reconnect()
		}
		n.lock.Unlock()
		return err
	}
	return nil
}

func (n *peer) callAsync(serviceMethod string, req, resp interface{}, done chan *rpc.Call) {
	if done != nil && cap(done) == 0 {
		log.Panic("cluster: RPC done channel is unbuffered")
	}
	n.lock.RLock()
	if !n.connected {
		call := &rpc.Call{
			ServiceMethod: serviceMethod,
			Args:          req,
			Reply:         resp,
			Error:         errors.New("cluster: peer '" + n.name + "' not connected"),
			Done:          done,
		}

		if done != nil {
			done <- call
		}
		n.lock.RUnlock()
		return
	}
	n.lock.RUnlock()
	myDone := make(chan *rpc.Call, 1)
	go func() {
		call := <-myDone
		if call.Error != nil {
			n.lock.Lock()
			if n.connected {
				n.endpoint.Close()
				n.connected = false
				n.lock.Unlock()
				go n.reconnect()
			} else {
				n.lock.Unlock()
			}
		}
		if done != nil {
			done <- call
		}
	}()
	n.endpoint.Go(serviceMethod, req, resp, myDone)
}
func (n *peer) callAsync2(serviceMethod string, req, resp interface{}, done chan *rpc.Call) *rpc.Call {
	if done != nil && cap(done) == 0 {
		log.Panic("cluster: RPC done channel is unbuffered")
	}
	n.lock.RLock()
	if !n.connected {
		call := &rpc.Call{
			ServiceMethod: serviceMethod,
			Args:          req,
			Reply:         resp,
			Error:         errors.New("cluster: peer '" + n.name + "' not connected"),
			Done:          done,
		}

		if done != nil {
			done <- call
		}
		n.lock.RUnlock()
		return call
	}
	n.lock.RUnlock()
	myDone := make(chan *rpc.Call, 1)
	go func() {
		call := <-myDone
		if call.Error != nil {
			n.lock.Lock()
			if n.connected {
				n.endpoint.Close()
				n.connected = false
				n.lock.Unlock()
				go n.reconnect()
			} else {
				n.lock.Unlock()
			}
		}
		if done != nil {
			done <- call
		}
	}()
	call := n.endpoint.Go(serviceMethod, req, resp, myDone)
	return call
}
