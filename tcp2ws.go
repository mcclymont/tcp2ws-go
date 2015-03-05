package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/url"
	"os"
)

const (
	CONN_TYPE   = "tcp"
	BUFFER_SIZE = 1024
)

func main() {
	rhost := flag.String("rhost", "REQUIRED", "URL of upstream websocket server. Format is 'ws://HOST:PORT/PATH'")
	lhost := flag.String("lhost", "localhost:3333", "TCP HOST:PORT to listen on")

	flag.Parse()
	if *rhost == "REQUIRED" {
		fmt.Println("Usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}
	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, *lhost)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + *lhost)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn, *rhost)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, rhost string) {
	fmt.Println("TCP connection received")
	defer func() { fmt.Println("Connection closed") }()
	defer conn.Close()

	ws_url, err := url.Parse(rhost)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	wsConn, err := net.Dial(CONN_TYPE, ws_url.Host)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer wsConn.Close()

	REQUEST_HEADERS := map[string][]string{
		"Sec-Websocket-Protocol": []string{"binary", "base64"},
	}
	ws, response, err := websocket.NewClient(wsConn, ws_url, REQUEST_HEADERS, BUFFER_SIZE, BUFFER_SIZE)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer ws.Close()

	protocol_arr := response.Header["Sec-Websocket-Protocol"]
	var protocol string
	if len(protocol_arr) > 0 {
		protocol = protocol_arr[0]
	}

	var messageType int
	if protocol == "base64" {
		messageType = websocket.TextMessage
	} else if protocol == "binary" {
		messageType = websocket.BinaryMessage
	} else {
		fmt.Println("Expected Sec-Websocket-Protocol header value of binary or base64, got:")
		fmt.Println(response.Header)
		return
	}

	fmt.Println("Websocket connection established, proxying...")
	var stopChan = make(chan int)
	go pipe_to_ws(conn, ws, messageType, stopChan)
	pipe_to_net(conn, ws, messageType, stopChan)
}

func pipe_to_net(netConn net.Conn, wsConn *websocket.Conn, protocolType int, stopChan chan int) {
	for {
		select {
		case <-stopChan:
			return
		default:
			{
				_, p, err := wsConn.ReadMessage()
				if err != nil {
					stopChan <- 1
					return
				}
				if protocolType == websocket.BinaryMessage {
					_, err = netConn.Write(p)
					if err != nil {
						stopChan <- 1
						return
					}
				} else if protocolType == websocket.TextMessage {
					data := make([]byte, base64.StdEncoding.DecodedLen(len(p)))
					n, err := base64.StdEncoding.Decode(data, p)
					if err != nil {
						stopChan <- 1
						return
					}
					_, err = netConn.Write(data[:n])
					if err != nil {
						stopChan <- 1
						return
					}
				}
			}
		}
	}
}

func pipe_to_ws(netConn net.Conn, wsConn *websocket.Conn, protocolType int, stopChan chan int) {
	for {
		select {
		case <-stopChan:
			return
		default:
			{
				data := make([]byte, BUFFER_SIZE)
				n, err := netConn.Read(data)
				if err != nil {
					stopChan <- 1
					return
				}

				if protocolType == websocket.BinaryMessage {
					err = wsConn.WriteMessage(protocolType, data[:n])
					if err != nil {
						stopChan <- 1
						return
					}
				} else if protocolType == websocket.TextMessage {
					encodedData := []byte(base64.StdEncoding.EncodeToString(data[:n]))
					err = wsConn.WriteMessage(protocolType, encodedData)
					if err != nil {
						stopChan <- 1
						return
					}
				}
			}
		}
	}
}
