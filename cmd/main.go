package main

import (
	"fitness-backend/internal" // Замени на свой путь к пакету
	"log"

	"github.com/gin-gonic/gin"
)

func main() {

	internal.InitDatabase()
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.POST("/register", internal.NewUser)
	r.POST("/login", internal.Login)

	api := r.Group("/api")
	api.Use(internal.AuthMiddleware())
	{

		api.GET("/profile", internal.GetProfile)
		api.PATCH("/profile/update", internal.UpdateProfile)
		api.GET("/workouts", internal.GetWorkouts)
		api.POST("/workouts/generate", internal.SetupWorkoutPlan)
		api.PATCH("/workouts/complete", internal.MarkWorkoutDone)
		api.GET("/stats", internal.GetStats)
	}

	log.Println("Сервер запущен на порту :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Не удалось запустить сервер: ", err)
	}
}
