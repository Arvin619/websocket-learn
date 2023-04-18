package main

import (
	"github.com/Arvin619/websocket-learn/chatroom"
)

func main() {
	c := chatroom.New(8080)
	c.Run()
}
