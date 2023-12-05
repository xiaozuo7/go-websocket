package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-websocket/controller"
	"go-websocket/global/variable"
	"go-websocket/util/ws/core"
	"net/http"
	"time"
)

func init() {
	variable.WebsocketHub = core.CreateHubFactory()
	if Wh, ok := variable.WebsocketHub.(*core.Hub); ok {
		go Wh.Run()
	}
}

func main() {
	r := gin.Default()

	// 设置跨域请求的响应头
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// 广播测试接口
	r.GET("/ping", func(c *gin.Context) {
		msg := fmt.Sprintf("hello, time: %v", time.Now().Format("2006-01-02 15:04:05"))
		notifyAllClient(msg)
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	// 通知某一个客户端
	r.GET("/send", func(c *gin.Context) {
		client := c.Query("client")
		onlineList := getClient(client)
		if len(onlineList) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"message": "no client",
			})
			return
		}
		notifyExpectClient("hello, client", onlineList)
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	r.GET("/ws", (&controller.Ws{}).OnOpen)
	_ = r.Run(":8080")
}

func notifyAllClient(message string) {
	variable.WebsocketHub.(*core.Hub).BroadCast <- []byte(message)
}

// 用list的原因是，一个用户可能会在多个地方登录，所以需要通知所有的客户端
func getClient(username string) []*core.Client {
	res := make([]*core.Client, 0)
	for client := range variable.WebsocketHub.(*core.Hub).Clients {
		if client.UserName == username {
			res = append(res, client)
		}
	}
	return res
}

// 通知指定客户端
func notifyExpectClient(message string, client []*core.Client) {
	for _, c := range client {
		err := c.SendMessage(1, message)
		if err != nil {
			fmt.Printf("消息发送错误：%v", err)
			_ = c.Conn.Close()
			delete(variable.WebsocketHub.(*core.Hub).Clients, c)
		}
	}

}
