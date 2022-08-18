package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	routes "github.com/leonardodelira/gojwt/routes"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8000"
	}

	router := gin.New()
	router.Use(gin.Logger())

	routes.AuthRoutes(router)
	routes.UserRoutes(router)
	router.GET("/test", func(c *gin.Context){
		c.JSON(200, gin.H{"sucess": "ok"})
	})
	router.Run(fmt.Sprintf(":%v", port))
}