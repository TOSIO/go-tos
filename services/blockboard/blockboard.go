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
	"fmt"
	"net"
	"time"

	"net/http"
	//"runtime"
	"sync"

	"errors"
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
		inChan: make(chan *wsMessage, 1000),
		outChan: make(chan *wsMessage, 1000),
		closeChan: make(chan byte),
		msg:SendData{
			Number:"",
			Status:"",
			MainBlock:"",
		},
		isClosed: false,
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
		return err
	}
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
	db.conn = conn
	// 处理器
	go db.procLoop()
	// 读协程
	go db.wsReadLoop()
	// 写协程
	go db.wsWriteLoop()
}

// Stop stops the data collection thread and the connection listener of the Blockboard.
// Implements the node.Service interface.
func (db *Blockboard) Stop() error {
	// Close the connection listener.
	var errs []error
	if err := db.listener.Close(); err != nil {
		errs = append(errs, err)
	}
	db.wsClose()
	var err error
	if len(errs) > 0 {
		err = fmt.Errorf("%v", errs)
	}
	return err
}

func (db *Blockboard)procLoop() {
	// 启动一个gouroutine发送心跳
	go func() {
		for {
			//组装发送数据
			db.msg.Number = GetSdagInfo(GetConnectNumber)
			db.msg.MainBlock =GetSdagInfo(GetMainBlockNumber)
			db.msg.Status = GetSdagInfo(GetSyncStatus)

			data, err := json.Marshal(db.msg)
			if err != nil {
				fmt.Printf(err.Error())
				return
			}
			time.Sleep(2 * time.Second)
			if err := db.wsWrite(websocket.TextMessage, data); err != nil {
				fmt.Println("heartbeat fail")
				db.wsClose()
				break
			}
		}
	}()

	// 这是一个同步处理模型（只是一个例子），如果希望并行处理可以每个请求一个gorutine，注意控制并发goroutine的数量!!!
	for {
		msg, err := db.wsRead()
		if err != nil {
			fmt.Println("read fail")
			break
		}
		fmt.Println(string(msg.data))
		err = db.wsWrite(msg.messageType, msg.data)
		if err != nil {
			fmt.Println("write fail")
			break
		}
	}
}

func (db *Blockboard)wsReadLoop() {
	for {
		// 读一个message
		msgType, data, err := db.conn.ReadMessage()
		if err != nil {
			goto error
		}
		req := &wsMessage{
			msgType,
			data,
		}
		// 放入请求队列
		select {
		case db.inChan <- req:
		case <- db.closeChan:
			goto closed
		}
	}
error:
	db.wsClose()
closed:
}

func (db *Blockboard)wsWriteLoop() {
	for {
		select {
		// 取一个应答
		case msg := <- db.outChan:
			// 写给websocket
			if err := db.conn.WriteMessage(msg.messageType, msg.data); err != nil {
				goto error
			}
		case <- db.closeChan:
			goto closed
		}
	}
error:
	db.wsClose()
closed:
}

func (db *Blockboard)wsWrite(messageType int, data []byte) error {
	select {
	case db.outChan <- &wsMessage{messageType, data,}:
	case <- db.closeChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (db *Blockboard)wsRead() (*wsMessage, error) {
	select {
	case msg := <- db.inChan:
		return msg, nil
	case <- db.closeChan:
	}
	return nil, errors.New("websocket closed")
}

func (db *Blockboard)wsClose() {
	db.conn.Close()
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if !db.isClosed {
		db.isClosed = true
		close(db.closeChan)
	}
}


