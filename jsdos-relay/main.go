package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var port string
var cert string
var key string

func init() {
	flag.StringVar(&cert, "c", "", ".cert file")
	flag.StringVar(&key, "k", "", ".key file")
	flag.StringVar(&port, "port", "1900", "Port to listen on")
	flag.Parse()
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
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "ok",
		"message": "pong",
	}
	json.NewEncoder(w).Encode(response)
}

var logprefix = "jsdos-relay"

func main() {
	log.Println(logprefix, "Listening on port", port)

	// http.HandleFunc("/ping", pingHandler)

	// http.HandleFunc("/ipx/", ipxWebSocket)
	// if len(cert) == 0 || len(key) == 0 {
	// 	log.Println(logprefix, ".cert or .key file is not provided, disabling TLS")
	// 	if err := http.ListenAndServe(":"+port, nil); err != nil {
	// 		log.Fatal(logprefix, err)
	// 	}
	// } else if err := http.ListenAndServeTLS(":"+port, cert, key, nil); err != nil {
	// 	log.Fatal(logprefix, err)
	// }

	hasKeyAndCert := len(cert) != 0 && len(key) != 0

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/ipx/", ipxWebSocket)

	var tlsCfg *tls.Config = nil
	if hasKeyAndCert {
		tlsCfg = &tls.Config{
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				keyPair, err := tls.LoadX509KeyPair(cert, key)
				if err != nil {
					return nil, err
				}
				return &keyPair, nil
			},
		}
	}

	srv := &http.Server{
		Addr:      fmt.Sprintf(":%s", port),
		Handler:   mux,
		TLSConfig: tlsCfg,
	}

	if !hasKeyAndCert {
		log.Println(logprefix, ".cert or .key file is not provided, disabling TLS")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(logprefix, err)
		}
	} else if err := srv.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(logprefix, err)
	}

	// run server on port "8443"
	log.Fatal()
}
