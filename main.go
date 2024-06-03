package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

type Config struct {
	WebSocketURL string `json:"websocket_url"`
}

type PM2Service struct {
	Name string `json:"name"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	// Serve the main HTML page
	http.HandleFunc("/", serveHome)

	// Serve configuration file
	http.HandleFunc("/config", handleConfig)

	// Handle WebSocket connections for logs
	http.HandleFunc("/logs", handleLogs)

	// Handle listing PM2 services
	http.HandleFunc("/services", handleServices)

	log.Println("Server started at :9192")
	log.Fatal(http.ListenAndServe(":9192", nil))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	htmlContent := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PM2 Log Streamer</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f0f0f0;
            margin: 0;
            padding: 0;
        }
        .container {
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background-color: #fff;
            box-shadow: 0 0 10px rgba(0,0,0,0.1);
            border-radius: 8px;
        }
        h1 {
            text-align: center;
            color: #333;
        }
        .log-container {
            height: 400px;
            overflow-y: scroll;
            background-color: #1e1e1e;
            color: #fff;
            padding: 10px;
            border-radius: 4px;
            font-family: monospace;
            white-space: pre-wrap;
        }
        .log-container div {
            padding: 2px 0;
        }
        .service-selection {
            margin: 20px 0;
            text-align: center;
        }
        .service-selection select {
            padding: 5px;
            font-size: 16px;
        }
        .service-selection button {
            padding: 5px 10px;
            font-size: 16px;
            margin-left: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>PM2 Log Streamer</h1>
        <div class="service-selection">
            <select id="service-select">
                <option value="all">All Services</option>
            </select>
            <button onclick="streamLogs()">Stream Logs</button>
        </div>
        <div id="log-container" class="log-container"></div>
    </div>
    <script>
        document.addEventListener("DOMContentLoaded", function() {
            fetch('/services')
                .then(response => response.json())
                .then(services => {
                    const serviceSelect = document.getElementById('service-select');
                    services.forEach(service => {
                        const option = document.createElement('option');
                        option.value = service.name;
                        option.textContent = service.name;
                        serviceSelect.appendChild(option);
                    });
                })
                .catch(error => console.error("Error fetching services:", error));
        });

        function streamLogs() {
            const logContainer = document.getElementById('log-container');
            const serviceSelect = document.getElementById('service-select');
            const selectedService = serviceSelect.value;

            logContainer.innerHTML = ''; // Clear existing logs

            fetch('/config')
                .then(response => response.json())
                .then(config => {
                    const ws = new WebSocket(config.websocket_url + '?service=' + selectedService);
                    ws.onopen = function() {
                        console.log("WebSocket connection opened");
                    };
                    ws.onmessage = function(event) {
                        const logMessage = document.createElement('div');
                        logMessage.textContent = event.data;
                        logContainer.appendChild(logMessage);
                        logContainer.scrollTop = logContainer.scrollHeight;
                    };
                    ws.onclose = function() {
                        console.log("WebSocket connection closed");
                    };
                    ws.onerror = function(error) {
                        console.log("WebSocket error:", error);
                    };
                })
                .catch(error => console.error("Error fetching config:", error));
        }
    </script>
</body>
</html>`
	w.Write([]byte(htmlContent))
}

// func serveHome(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "text/html")
// 	htmlContent := `
// <!DOCTYPE html>
// <html lang="en">
// <head>
//     <meta charset="UTF-8">
//     <meta name="viewport" content="width=device-width, initial-scale=1.0">
//     <title>PM2 Log Streamer</title>
//     <style>
//         body {
//             font-family: Arial, sans-serif;
//             background-color: #f0f0f0;
//             margin: 0;
//             padding: 0;
//         }
//         .container {
//             max-width: 800px;
//             margin: 50px auto;
//             padding: 20px;
//             background-color: #fff;
//             box-shadow: 0 0 10px rgba(0,0,0,0.1);
//             border-radius: 8px;
//         }
//         h1 {
//             text-align: center;
//             color: #333;
//         }
//         .log-container {
//             height: 400px;
//             overflow-y: scroll;
//             background-color: #1e1e1e;
//             color: #fff;
//             padding: 10px;
//             border-radius: 4px;
//             font-family: monospace;
//             white-space: pre-wrap;
//         }
//         .log-container div {
//             padding: 2px 0;
//         }
//         .services {
//             margin: 20px 0;
//         }
//         .services a {
//             display: block;
//             margin: 5px 0;
//             color: #007bff;
//             text-decoration: none;
//         }
//         .services a:hover {
//             text-decoration: underline;
//         }
//     </style>
// </head>
// <body>
//     <div class="container">
//         <h1>PM2 Log Streamer</h1>
//         <div class="services">
//             <a href="#" onclick="streamLogs('all')">Stream All Logs</a>
//             <div id="service-list"></div>
//         </div>
//         <div id="log-container" class="log-container"></div>
//     </div>
//     <script>
//         document.addEventListener("DOMContentLoaded", function() {
//             fetch('/services')
//                 .then(response => response.json())
//                 .then(services => {
//                     const serviceList = document.getElementById('service-list');
//                     services.forEach(service => {
//                         const serviceLink = document.createElement('a');
//                         serviceLink.href = "#";
//                         serviceLink.textContent = service.name;
//                         serviceLink.onclick = () => streamLogs(service.name);
//                         serviceList.appendChild(serviceLink);
//                     });
//                 })
//                 .catch(error => console.error("Error fetching services:", error));
//         });

//         function streamLogs(serviceName) {
//             const logContainer = document.getElementById('log-container');
//             logContainer.innerHTML = ''; // Clear existing logs

//             fetch('/config')
//                 .then(response => response.json())
//                 .then(config => {
//                     const ws = new WebSocket(config.websocket_url + '?service=' + serviceName);
//                     ws.onopen = function() {
//                         console.log("WebSocket connection opened");
//                     };
//                     ws.onmessage = function(event) {
//                         const logMessage = document.createElement('div');
//                         logMessage.textContent = event.data;
//                         logContainer.appendChild(logMessage);
//                         logContainer.scrollTop = logContainer.scrollHeight;
//                     };
//                     ws.onclose = function() {
//                         console.log("WebSocket connection closed");
//                     };
//                     ws.onerror = function(error) {
//                         console.log("WebSocket error:", error);
//                     };
//                 })
//                 .catch(error => console.error("Error fetching config:", error));
//         }
//     </script>
// </body>
// </html>`
// 	w.Write([]byte(htmlContent))
// }

func handleConfig(w http.ResponseWriter, r *http.Request) {
	config := Config{
		WebSocketURL: os.Getenv("WEBSOCKET_URL"),
	}
	if config.WebSocketURL == "" {
		config.WebSocketURL = "ws://localhost:9192/logs"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func handleServices(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("pm2", "list")
	output, err := cmd.Output()
	if err != nil {
		log.Println("Failed to list PM2 services:", err)
		http.Error(w, "Could not list PM2 services", http.StatusInternalServerError)
		return
	}

	var services []PM2Service
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "│") && !strings.Contains(line, "App name") && !strings.Contains(line, "│ id") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				services = append(services, PM2Service{Name: fields[3]})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade failed:", err)
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	service := r.URL.Query().Get("service")
	var cmd *exec.Cmd

	if service == "all" {
		cmd = exec.Command("pm2", "logs", "--raw")
	} else {
		cmd = exec.Command("pm2", "logs", service, "--raw")
	}

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
