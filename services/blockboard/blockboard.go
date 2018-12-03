// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package blockboard

//go:generate yarn --cwd ./assets install
//go:generate yarn --cwd ./assets build
//go:generate go-bindata -nometadata -o assets.go -prefix assets -nocompress -pkg Blockboard assets/index.html assets/bundle.js
//go:generate sh -c "sed 's#var _bundleJs#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate sh -c "sed 's#var _indexHtml#//nolint:misspell\\\n&#' assets.go > assets.go.tmp && mv assets.go.tmp assets.go"
//go:generate gofmt -w -s assets.go

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"net/http"
	//"runtime"
	"sync"

	//"errors"
	//"github.com/TOSIO/go-tos/app/sendTx/httpSend"
	"github.com/TOSIO/go-tos/devbase/log"
	//"github.com/TOSIO/go-tos/sdag"
	"github.com/TOSIO/go-tos/services/p2p"
	"github.com/TOSIO/go-tos/services/rpc"
	"github.com/gorilla/websocket"
)

// http升级websocket协议的配置
var wsUpgrader = websocket.Upgrader{
	// 允许所有CORS跨域请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Blockboard contains the Blockboard internals.
type Blockboard struct {
	config *Config
	listener   net.Listener
	logdir     string
	wsConn	*wsConnection
}

// 客户端连接
type wsConnection struct {
	conn *websocket.Conn // 底层websocket
	inChan chan *wsMessage	// 读队列
	outChan chan *wsMessage // 写队列

	mutex sync.Mutex	// 避免重复关闭管道
	isClosed bool
	closeChan chan byte  // 关闭通知
	msg SendData
}
//发送的数据格式
type SendData struct {
	Number string `json:"number"`
	Status string `json:"status"`
	MainBlock string `json:"main_block"`
	SyncPercent string `json:"sync_percent"`
}


// 客户端读写消息
type wsMessage struct {
	messageType int
	data []byte
}


// New creates a new Blockboard instance with the given configuration.
func New(config *Config, commit string, logdir string) *Blockboard {

	return &Blockboard{
		config: config,
		logdir: logdir,
	}
}



// Protocols implements the node.Service interface.
func (db *Blockboard) Protocols() []p2p.Protocol { return nil }

// APIs implements the node.Service interface.
func (db *Blockboard) APIs() []rpc.API { return nil }

// Start starts the data collection thread and the listening server of the Blockboard.
// Implements the node.Service interface.
func (db *Blockboard) Start(server *p2p.Server) error  {
	log.Info("Starting Blockboard")

	http.HandleFunc("/api", db.wsHandler)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", db.config.Host, db.config.Port))
	if err != nil {
		log.Debug("Error start blockboard","err",err)
		return err
	}
	log.Debug("Starting Blockboard  ok")
	db.listener = listener

	go http.Serve(listener, nil)

	return nil

}


func (db *Blockboard)wsHandler(resp http.ResponseWriter, req *http.Request) {
	// 应答客户端告知升级连接为websocket
	conn, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		return
	}
	wsConn := &wsConnection{
		conn: conn,
		inChan: make(chan *wsMessage, 1000),
		outChan: make(chan *wsMessage, 1000),
		closeChan: make(chan byte),
		isClosed: false,
	}
	db.wsConn =wsConn
	// 处理器
	go db.wsConn.procLoop()
	// 读协程
	go db.wsConn.wsReadLoop()
	// 写协程
	go db.wsConn.wsWriteLoop()
}

// Stop stops the data collection thread and the connection listener of the Blockboard.
// Implements the node.Service interface.
func (db *Blockboard) Stop() error {
	// Close the connection listener.
	var errs []error
	if err := db.listener.Close(); err != nil {
		errs = append(errs, err)
	}
	db.wsConn.wsClose()
	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("%v", errs)
	}
	return err
}

func (ws *wsConnection)procLoop() {
	// 启动一个gouroutine发送心跳
	go func() {
		for {
			//组装发送数据
			ws.msg.Number = GetSdagInfo(GetConnectNumber)
			ws.msg.Status = GetSdagInfo(GetSyncStatus)
			ws.msg.MainBlock =GetSdagInfo(GetMainBlockNumber)
			ws.msg.SyncPercent = GetSdagInfo(GetProgressPercent)


			data, err := json.Marshal(ws.msg)
			if err != nil {
				log.Debug("Error blockboard procLoop","err",err)
				return
			}
			time.Sleep(2 * time.Second)
			if err := ws.wsWrite(websocket.TextMessage, data); err != nil {
				log.Debug("Error Blockboard procLoop db.wsWrite","err",err)
				ws.wsClose()
				break
			}
		}
	}()

	// 这是一个同步处理模型（只是一个例子），如果希望并行处理可以每个请求一个gorutine，注意控制并发goroutine的数量!!!
	for {
		msg, err := ws.wsRead()
		if err != nil {
			log.Debug("Error blockboard read fail")
			break
		}
		err = ws.wsWrite(msg.messageType, msg.data)
		if err != nil {
			log.Debug("Error blockboard write fail")
			break
		}
	}
}

func (ws *wsConnection)wsReadLoop() {
	for {
		// 读一个message
		msgType, data, err := ws.conn.ReadMessage()
		if err != nil {
			goto error
		}
		req := &wsMessage{
			msgType,
			data,
		}
		// 放入请求队列
		select {
		case ws.inChan <- req:
		case <- ws.closeChan:
			log.Debug("Error Blockboard wsReadLoop closeChan")
			goto closed
		}
	}
error:
	ws.wsClose()
closed:
}

func (ws *wsConnection)wsWriteLoop() {
	for {
		select {
		// 取一个应答
		case msg := <- ws.outChan:
			// 写给websocket
			if err := ws.conn.WriteMessage(msg.messageType, msg.data); err != nil {
				log.Debug("Error Blockboard wsWriteLoop WriteMessage","err",err)
				goto error
			}
		case <- ws.closeChan:
			log.Debug("Error Blockboard wsWriteLoop WriteMessage closeChan")
			goto closed
		}
	}
error:
	ws.wsClose()
closed:
}

func (ws *wsConnection)wsWrite(messageType int, data []byte) error {
	select {
	case ws.outChan <- &wsMessage{messageType, data,}:
	case <- ws.closeChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (ws *wsConnection)wsRead() (*wsMessage, error) {
	select {
	case msg := <- ws.inChan:
		return msg, nil
	case <- ws.closeChan:
	}
	return nil, errors.New("websocket closed")
}

func (ws *wsConnection)wsClose() {
	ws.conn.Close()
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if !ws.isClosed {
		ws.isClosed = true
		close(ws.closeChan)
	}
}


