package core

import (
	"fmt"
)

type Hub struct {
	//上线注册
	Register chan *Client
	//下线注销
	UnRegister chan *Client
	//所有在线客户端的内存地址
	Clients map[*Client]bool
	//广播消息
	BroadCast chan []byte
}

func CreateHubFactory() *Hub {
	return &Hub{
		Register:   make(chan *Client),
		UnRegister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		BroadCast:  make(chan []byte),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
		case client := <-h.UnRegister:
			if _, ok := h.Clients[client]; ok {
				_ = client.Conn.Close()
				delete(h.Clients, client)
			}
		case message := <-h.BroadCast:
			for client := range h.Clients {
				err := client.SendMessage(1, string(message))
				if err != nil {
					fmt.Printf("消息发送错误：%v", err)
					if _, ok := h.Clients[client]; ok {
						_ = client.Conn.Close()
						delete(h.Clients, client)
					}
				}
			}

		}

	}
}
