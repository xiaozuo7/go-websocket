package core

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-websocket/global/variable"
	"net/http"
	"sync"
	"time"
)

// 待提取配置
const (
	writeReadBufferSize   = 20480 // 读写缓冲区分配字节，大概能存储 6800 多一点的文字
	maxMessageSize        = 65535 // 从消息管道读取消息的最大字节
	pingPeriod            = 20    // 心跳包频率，单位：秒
	heartbeatFailMaxTimes = 3     // 允许心跳失败的最大次数（默认设置为PingPeriod=30秒检测一次，连续4次没有心跳就会清除后端在线信息）
	readDeadline          = 100   // 客户端在线情况下，正常的业务消息间隔秒数必须小于该值，否则服务器将会主动断开，该值不能小于心跳频率*允许失败次数,单位：秒。 0 表示不设限制，即服务器不主动断开不发送任何消息的在线客户端，但会消耗服务器资源
	writeDeadline         = 35    // 消息单次写入超时时间，单位：秒

)

type Client struct {
	Hub                *Hub            // 负责处理客户端注册、注销、在线管理
	Conn               *websocket.Conn // 一个ws连接
	Send               chan []byte     // 一个ws连接存储自己的消息管道
	PingPeriod         time.Duration
	ReadDeadline       time.Duration
	WriteDeadline      time.Duration
	HeartbeatFailTimes int
	State              uint8 // ws状态，1=ok；2=出错、掉线等
	sync.RWMutex
	ClientMoreParams
}

// ClientMoreParams 这里追加一个结构体，方便开发者在成功上线后，可以自定义追加更多字段信息
type ClientMoreParams struct {
	UserName string
}

// OnOpen 处理握手+协议升级
func (c *Client) OnOpen(context *gin.Context) (*Client, bool) {
	// 1.升级连接,从http--->websocket
	defer func() {
		err := recover()
		if err != nil {
			fmt.Printf("websocket 升级失败：%v", err)
		}
	}()
	var upGrader = websocket.Upgrader{
		ReadBufferSize:  writeReadBufferSize, //
		WriteBufferSize: writeReadBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 2.将http协议升级到websocket协议.初始化一个有效的websocket长连接客户端
	if wsConn, err := upGrader.Upgrade(context.Writer, context.Request, nil); err != nil {
		return nil, false
	} else {
		if wsHub, ok := variable.WebsocketHub.(*Hub); ok {
			c.Hub = wsHub
		}
		c.Conn = wsConn
		c.Send = make(chan []byte, writeReadBufferSize)
		c.PingPeriod = time.Second * pingPeriod
		c.ReadDeadline = time.Second * readDeadline
		c.WriteDeadline = time.Second * writeDeadline

		if err := c.SendMessage(websocket.TextMessage, variable.WebsocketHandshakeSuccess); err != nil {
			fmt.Printf("消息发送错误：%v", err)

		}
		c.Conn.SetReadLimit(maxMessageSize) // 设置最大读取长度
		c.Hub.Register <- c
		c.State = 1
		return c, true
	}

}

// ReadPump 实时接收消息
func (c *Client) ReadPump(callbackOnMessage func(messageType int, receivedData []byte), callbackOnError func(err error), callbackOnClose func()) {
	// 回调 onclose 事件
	defer func() {
		err := recover()
		if err != nil {
			fmt.Println(err)
			return
		}
		callbackOnClose()
	}()

	// OnMessage事件
	for {
		if c.State == 1 {
			mt, bReceivedData, err := c.Conn.ReadMessage()
			if err == nil {
				callbackOnMessage(mt, bReceivedData)
			} else {
				// OnError事件读（消息出错)
				callbackOnError(err)
				break
			}
		} else {
			// OnError事件(状态不可用，一般是程序事先检测到双方无法进行通信，进行的回调)
			callbackOnError(errors.New("websocket  state 状态已经不可用(掉线、卡死等原因，造成双方无法进行数据交互)"))
			break
		}

	}
}

// SendMessage 发送消息，请统一调用本函数进行发送
// 消息发送时增加互斥锁，加强并发情况下程序稳定性
// 提醒：开发者发送消息时，不要调用 c.Conn.WriteMessage(messageType, []byte(message)) 直接发送消息
func (c *Client) SendMessage(messageType int, message string) error {
	c.Lock()
	defer func() {
		c.Unlock()
	}()
	// 发送消息时，必须设置本次消息的最大允许时长(秒)
	if err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteDeadline)); err != nil {
		return err
	}
	if err := c.Conn.WriteMessage(messageType, []byte(message)); err != nil {
		return err
	} else {
		return nil
	}
}

// Heartbeat 按照websocket标准协议实现隐式心跳,Server端向Client远端发送ping格式数据包,浏览器收到ping标准格式，自动将消息原路返回给服务器
func (c *Client) Heartbeat() {
	//  1. 设置一个时钟，周期性的向client远端发送心跳数据包
	ticker := time.NewTicker(c.PingPeriod)
	defer func() {
		err := recover()
		if err != nil {
			return
		}
		ticker.Stop() // 停止该client的心跳检测
	}()
	//2.浏览器收到服务器的ping格式消息，会自动响应pong消息，将服务器消息原路返回过来
	if c.ReadDeadline == 0 {
		_ = c.Conn.SetReadDeadline(time.Time{})
	} else {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.ReadDeadline))
	}
	c.Conn.SetPongHandler(func(receivedPong string) error {
		if c.ReadDeadline > time.Nanosecond {
			_ = c.Conn.SetReadDeadline(time.Now().Add(c.ReadDeadline))
		} else {
			_ = c.Conn.SetReadDeadline(time.Time{})
		}
		return nil
	})
	//3.自动心跳数据
	for {
		select {
		case <-ticker.C:
			if c.State == 1 {
				if err := c.SendMessage(websocket.PingMessage, variable.WebsocketServerPingMsg); err != nil {
					c.HeartbeatFailTimes++
					if c.HeartbeatFailTimes > heartbeatFailMaxTimes {
						c.State = 2
						// log记录 心跳失败记录超过最大上限
						return
					}
				} else {
					if c.HeartbeatFailTimes > 0 {
						c.HeartbeatFailTimes--
					}
				}
			} else {
				return
			}

		}
	}
}
