package tcp2ws

import (
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/url"
)

const (
	CONN_TYPE   = "tcp"
	BUFFER_SIZE = 1024
)

// If forever is true, it will run in an infinite loop accepting new
// connections and proxying them to rhost.
// If it is false, it will stop listening on listenHost after
// the first connection is established.
func Proxy(forever bool, listenHost string, remoteHost string) error {
	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, listenHost)
	if err != nil {
		return fmt.Errorf("Error listening:", err.Error())
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + listenHost)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("Error accepting: ", err.Error())
		}
		if forever {
			go handleRequest(conn, remoteHost)
		} else {
			return handleRequest(conn, remoteHost)
		}
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn, rhost string) error {
	fmt.Println("TCP connection received")
	defer func() { fmt.Println("Connection closed") }()
	defer conn.Close()

	ws_url, err := url.Parse(rhost)
	if err != nil {
		return err
	}

	wsConn, err := net.Dial(CONN_TYPE, ws_url.Host)
	if err != nil {
		return err
	}
	defer wsConn.Close()

	REQUEST_HEADERS := map[string][]string{
		"Sec-WebSocket-Protocol": []string{"binary", "base64"},
		"Sec-WebSocket-Version":  []string{"13"},
	}
	ws, response, err := websocket.NewClient(wsConn, ws_url, REQUEST_HEADERS, BUFFER_SIZE, BUFFER_SIZE)
	if err != nil {
		return err
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
		return fmt.Errorf("Expected Sec-Websocket-Protocol header value of binary or base64, got:\n%s", response.Header)
	}

	fmt.Println("Websocket connection established, proxying...")
	var stopChan = make(chan int)
	go pipe_to_ws(conn, ws, messageType, stopChan)
	pipe_to_net(conn, ws, messageType, stopChan)
	return nil
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
