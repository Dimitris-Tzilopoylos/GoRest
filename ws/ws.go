package ws

import (
	"net/http"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

var allowOriginFunc = func(r *http.Request) bool {
	return true
}

type WebSocketServer struct {
	Server     *socketio.Server
	namespaces map[string][]string
}

func NewWebSocketServer() *WebSocketServer {
	serverOptions := &engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
	}
	server := &WebSocketServer{Server: socketio.NewServer(serverOptions), namespaces: make(map[string][]string)}

	return server
}

func (ws *WebSocketServer) Connect(namespace string, handler func(socketio.Conn) error) {
	if _, ok := ws.namespaces[namespace]; !ok {
		ws.namespaces[namespace] = make([]string, 0)
	}

	ws.Server.OnConnect(namespace, handler)
}

func (ws *WebSocketServer) DisConnect(namespace string, handler func(socketio.Conn, string)) {
	ws.Server.OnDisconnect(namespace, handler)
}

func (ws *WebSocketServer) On(namespace string, event string, handler func(socketio.Conn, any) error) {
	ws.namespaces[namespace] = append(ws.namespaces[namespace], event)
	ws.Server.OnEvent(namespace, event, handler)
}

func (ws *WebSocketServer) RunServer() {
	go ws.Server.Serve()
}

func (ws *WebSocketServer) Close() {
	ws.Server.Close()
}
