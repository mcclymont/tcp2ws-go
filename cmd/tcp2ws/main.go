package main

import (
  "github.com/mcclymont/tcp2ws-go"
  "fmt"
  "flag"
  "os"
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
	err := tcp2ws.Proxy(true, *lhost, *rhost)
	if err != nil {
		fmt.Println(err)
	}
}
