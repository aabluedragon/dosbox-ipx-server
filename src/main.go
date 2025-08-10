package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"

	"os"

	"github.com/gorilla/websocket"
)

var port string

func init() {
	var portFlag string
	flag.StringVar(&portFlag, "port", "", "Port to listen on")
	flag.Parse()
	if portFlag != "" {
		port = portFlag
	} else if p := os.Getenv("PORT"); p != "" {
		port = p
	} else {
		port = "1900"
	}
}

var upgrader = websocket.Upgrader{
	Subprotocols: []string{"binary"},
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var ipxHandler = &IpxHandler{
	serverAddress: "127.0.0.1:" + port,
}

func getRoom(r *http.Request) string {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		return ""
	}
	if parts[1] != "ipx" {
		return ""
	}
	return parts[2]
}

func ipxWebSocket(w http.ResponseWriter, r *http.Request) {
	room := getRoom(r)
	if len(room) == 0 {
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	ipxHandler.OnConnect(conn, room)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		ipxHandler.OnMessage(conn, room, data)
	}

	ipxHandler.OnClose(conn, room)
	conn.Close()
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "ok",
		"message": "pong",
	}
	json.NewEncoder(w).Encode(response)
}

var cert string
var key string

func main() {
	log.Println("Listening on port", port)

	flag.StringVar(&cert, "c", "", ".cert file")
	flag.StringVar(&key, "k", "", ".key file")
	flag.Parse()

	http.HandleFunc("/ping", pingHandler)

	http.HandleFunc("/ipx/", ipxWebSocket)
	if len(cert) == 0 || len(key) == 0 {
		log.Println(".cert or .key file is not provided, disabling TLS")
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal(err)
		}
	} else if err := http.ListenAndServeTLS(":"+port, cert, key, nil); err != nil {
		log.Fatal(err)
	}
}
