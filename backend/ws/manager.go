package ws

import (
	"net/http"
	"github.com/gorilla/websocket"
)

var RequestAuthorizer func(*http.Request) error

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
