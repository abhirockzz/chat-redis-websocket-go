package chat

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

// Peers maps a chat user to the websocket connection (pointer)
var Peers map[string]*websocket.Conn

func init() {
	Peers = map[string]*websocket.Conn{}
}

// ChatSession represents a connected/active chat user
type ChatSession struct {
	user string
	peer *websocket.Conn
}

// NewChatSession returns a new ChatSession
func NewChatSession(user string, peer *websocket.Conn) *ChatSession {

	return &ChatSession{user: user, peer: peer}
}

const usernameHasBeenTaken = "username %s is already taken. please retry with a different name"
const retryMessage = "failed to connect. please try again"
const welcome = "Welcome %s!"
const joined = "%s: has joined the chat!"
const chat = "%s: %s"
const left = "%s: has left the chat!"

// Start starts the chat by reading messages sent by the peer and broadcasting the to redis pub-sub channel
func (s *ChatSession) Start() {
	usernameTaken, err := CheckUserExists(s.user)

	if err != nil {
		log.Println("unable to determine whether user exists -", s.user)
		s.notifyPeer(retryMessage)
		s.peer.Close()
		return
	}

	if usernameTaken {
		msg := fmt.Sprintf(usernameHasBeenTaken, s.user)
		s.peer.WriteMessage(websocket.TextMessage, []byte(msg))
		s.peer.Close()
		return
	}

	err = CreateUser(s.user)
	if err != nil {
		log.Println("failed to add user to list of active chat users", s.user)
		s.notifyPeer(retryMessage)
		s.peer.Close()
		return
	}
	Peers[s.user] = s.peer

	s.notifyPeer(fmt.Sprintf(welcome, s.user))
	SendToChannel(fmt.Sprintf(joined, s.user))

	/*
		this go-routine will exit when:
		(1) the user disconnects from chat manually
		(2) the app is closed
	*/
	go func() {
		log.Println("user joined", s.user)
		for {
			_, msg, err := s.peer.ReadMessage()
			if err != nil {
				_, ok := err.(*websocket.CloseError)
				if ok {
					log.Println("connection closed by user")
					s.disconnect()
				}
				return
			}
			SendToChannel(fmt.Sprintf(chat, s.user, string(msg)))
		}
	}()
}

func (s *ChatSession) notifyPeer(msg string) {
	err := s.peer.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		log.Println("failed to write message", err)
	}
}

// Invoked when the user disconnects (websocket connection is closed). It performs cleanup activities
func (s *ChatSession) disconnect() {
	//remove user from SET
	RemoveUser(s.user)

	//notify other users that this user has left
	SendToChannel(fmt.Sprintf(left, s.user))

	//close websocket
	s.peer.Close()

	//remove from Peers
	delete(Peers, s.user)
}
