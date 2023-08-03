package ws

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/synw/altiplano/goinfer/state"
)

var clients = make(map[*websocket.Conn]bool)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type TokenMsg struct {
	Msg    string `json:"msg"`
	Ntoken int    `json:"nToken"`
}

func RunWs() {
	if state.IsVerbose {
		fmt.Println("Starting the websockets server")
	}
	http.HandleFunc("/ws", handleConnections)
	err := http.ListenAndServe(":5142", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	//defer ws.Close()
	clients[ws] = true
}

func SendToken(msg string, i int) {
	message := TokenMsg{
		Msg:    msg,
		Ntoken: i,
	}
	for client := range clients {
		err := client.WriteJSON(&message)
		if err != nil {
			log.Printf("error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}
