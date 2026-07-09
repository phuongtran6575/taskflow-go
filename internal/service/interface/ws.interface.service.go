package _interface

type WebSocketHub interface {
	BroadcastToRoom(room, event string, data interface{})
}
