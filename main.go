package main

import (
	"bufio"
	"log"
	"net/http"
	"os/exec"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func main() {
	// Serve static files from the /static directory
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Handle WebSocket connections for logs
	http.HandleFunc("/logs", handleLogs)

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade failed:", err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Execute pm2 logs command
	cmd := exec.Command("pm2", "logs", "--raw")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Failed to get stdout pipe:", err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		if err := cmd.Start(); err != nil {
			log.Println("Command start failed:", err)
			return
		}
	}()

	for scanner.Scan() {
		message := scanner.Bytes()
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("Write message failed:", err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Scanner error:", err)
	}

	if err := cmd.Wait(); err != nil {
		log.Println("Command wait failed:", err)
	}
}
