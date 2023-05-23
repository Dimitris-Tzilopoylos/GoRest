package engine

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	Id   string `json:"id"`
	Auth jwt.MapClaims
	Conn *websocket.Conn
}

type Room struct {
	Id      string
	Clients []*Client
}

type Rooms []Room

type Message struct {
	Event string `json:"event"`
	Data  any    `json:data"`
}

func (r *Router) NewClient(conn *websocket.Conn, auth jwt.MapClaims) *Client {
	clientId := uuid.New().String()
	client := &Client{Conn: conn, Id: clientId, Auth: auth}
	r.Rooms[client] = true
	return client
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 5 * time.Second,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func SendMessage(client *Client, data any) {

}

func (r *Router) WSHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		log.Println(err)
		return
	}
	auth := GetAuth(req)
	client := r.NewClient(conn, auth)
	defer client.Conn.Close()
	defer func() {
		delete(r.Rooms, client)
	}()
	defer client.Conn.Close()
	for {
		messageType, msg, err := client.Conn.ReadMessage()
		if messageType == websocket.CloseMessage {
			log.Println("WebSocket connection closed:")
			break
		}
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Println("WebSocket connection closed:", err)
				return
			}
			fmt.Println(err)
			break
		}

		fmt.Println(msg, messageType)
	}
}
