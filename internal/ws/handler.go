package ws

import (
	"net/http"

	"TaskFlow-Go/internal/helper"
	"TaskFlow-Go/internal/shared/appresponse"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WSHandler(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			appresponse.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token")
			c.Abort()
			return
		}

		claims, err := helper.ValidateAccessToken(tokenStr)
		if err != nil {
			appresponse.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			c.Abort()
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := NewClient(hub, conn, claims.UserID)
		hub.register <- client

		go client.writePump()
		client.readPump()
	}
}

func WSProjectHandler(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			appresponse.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token")
			c.Abort()
			return
		}

		claims, err := helper.ValidateAccessToken(tokenStr)
		if err != nil {
			appresponse.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			c.Abort()
			return
		}

		projectID := c.Param("project_id")
		if projectID == "" {
			appresponse.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "Missing project_id")
			c.Abort()
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := NewClient(hub, conn, claims.UserID)
		hub.register <- client

		hub.join <- &RoomAction{client: client, room: projectID}

		go client.writePump()
		client.readPump()
	}
}
