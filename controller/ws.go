package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	websocket "go-websocket/services/websockt"
)

type Ws struct {
}

// OnOpen 主要解决握手+协议升级
func (w *Ws) OnOpen(context *gin.Context) {
	ws, ok := (&websocket.Ws{}).OnOpen(context)
	if !ok {
		fmt.Printf("websocket open阶段初始化失败")
		return
	}
	ws.OnMessage()
}
