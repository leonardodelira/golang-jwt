package routes

import (
	"github.com/gin-gonic/gin"
	controllers "github.com/leonardodelira/gojwt/controllers"
)

func AuthRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.POST("users/signup", controllers.SignUp())
	incomingRoutes.POST("users/login", controllers.Login())
}