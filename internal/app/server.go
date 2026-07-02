package app

import "github.com/gin-gonic/gin"

func NewRouter(container *Container) *gin.Engine {
	r := gin.Default()
	api := r.Group("/api/v1")
	container.SetupRoutes(api)
	return r
}

func StartServer(container *Container, addr ...string) error {
	router := NewRouter(container)
	port := ":8080"
	if len(addr) > 0 {
		port = addr[0]
	}
	return router.Run(port)
}
