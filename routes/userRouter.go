package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/leonardodelira/gojwt/controllers"
	"github.com/leonardodelira/gojwt/middlewares"
)

func UserRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middlewares.Authenticate())
	incomingRoutes.GET("/users", controllers.GetUsers())
	incomingRoutes.GET("/users/:userId", controllers.GetUser())
}