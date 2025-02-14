package main

import (
	"bpl/config"
	"bpl/controller"
	"bpl/docs"
	"bpl/repository"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           BPL Backend API
// @version         2.0
// @description     This is the backend API for the BPL project.

// @contact.name   	Liberator
// @contact.email 	Liberatorist@gmail.com

// @securityDefinitions.basic  BasicAuth

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	db, err := config.InitDB()
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(
		&repository.ScoringCategory{},
		&repository.Objective{},
		&repository.Condition{},
		&repository.Event{},
		&repository.Team{},
		&repository.User{},
		&repository.TeamUser{},
		&repository.StashChange{},
		&repository.ObjectiveMatch{},
		&repository.Submission{},
		&repository.ClientCredentials{},
		&repository.Signup{},
		&repository.Oauth{},
		&repository.KafkaConsumer{},
	)

	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://localhost"},
		AllowMethods:     []string{"GET", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	docs.SwaggerInfo.BasePath = "/api"
	r.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	controller.SetRoutes(r)
	r.Run(":8000")
}
