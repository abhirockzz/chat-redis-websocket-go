package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/abhirockzz/redis-chat-go/chat"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
var redisHost string
var redisPassword string

const port = "8080"

func init() {
	redisHost = os.Getenv("REDIS_HOST")
	if redisHost == "" {
		log.Fatal("missing REDIS_HOST env var")
	}

	redisPassword = os.Getenv("REDIS_PASSWORD")
	if redisPassword == "" {
		log.Fatal("missing REDIS_PASSWORD env var")
	}
}

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("you are good to go!"))
	})
	http.Handle("/chat/", http.HandlerFunc(websocketHandler))
	server := http.Server{Addr: ":" + port, Handler: nil}
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", err)
		}
	}()

	exit := make(chan os.Signal)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGINT)
	<-exit

	log.Println("exit signalled")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chat.Cleanup()
	server.Shutdown(ctx)

	log.Println("chat app exited")
}

func websocketHandler(rw http.ResponseWriter, req *http.Request) {
	user := strings.TrimPrefix(req.URL.Path, "/chat/")

	peer, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		log.Fatal("websocket conn failed", err)
	}

	chatSession := chat.NewChatSession(user, peer)
	chatSession.Start()
}
