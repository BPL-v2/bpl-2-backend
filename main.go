package main

import (
	"bpl/config"
	"bpl/controller"
	"bpl/docs"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

// @title           BPL Backend API
// @version         2.0
// @description     This is the backend API for the BPL project.

// @contact.name   	Liberator
// @contact.email 	Liberatorist@gmail.com

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	db, err := config.InitDB(
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("DATABASE_NAME"),
	)
	if err != nil {
		panic(err)
	}
	_ = db
	// err = db.AutoMigrate(
	// 	&repository.ScoringCategory{},
	// 	&repository.Objective{},
	// 	&repository.Condition{},
	// 	&repository.Event{},
	// 	&repository.Team{},
	// 	&repository.User{},
	// 	&repository.TeamUser{},
	// 	&repository.StashChange{},
	// 	&repository.ObjectiveMatch{},
	// 	&repository.Submission{},
	// 	&repository.ClientCredentials{},
	// 	&repository.Signup{},
	// 	&repository.Oauth{},
	// 	&repository.KafkaConsumer{},
	// 	&repository.RecurringJob{},
	// 	&repository.LadderEntry{},
	// 	&repository.Character{},
	// 	&repository.Atlas{},
	// 	&repository.TeamSuggestion{},
	// )

	// if err != nil {
	// 	panic(err)
	// }

	r := gin.Default()
	r.Use(gin.Recovery())
	corsConfigGetOptions := cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		MaxAge:       12 * time.Hour,
	}
	corsConfigOtherMethods := cors.Config{
		AllowOrigins: []string{
			"https://bpl-2.netlify.app",
			"https://v2202503259898322516.goodsrv.de",
			"http://localhost",
			"http://localhost:3000",
		},
		AllowMethods:     []string{"POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	r.Use(func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
			cors.New(corsConfigGetOptions)(c)
		} else {
			cors.New(corsConfigOtherMethods)(c)
		}
	})

	addMetrics(r)
	addDocs(r)

	controller.SetRoutes(r)
	r.Run(":8000")
}

func addMetrics(r *gin.Engine) {
	p := ginprometheus.NewPrometheus("gin")
	re := regexp.MustCompile(`\d+`)
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := strings.Split(c.Request.URL.String(), "?")[0]
		url = strings.ReplaceAll(url, "current", "?")
		url = re.ReplaceAllString(url, "?")
		return strings.TrimPrefix(url, "/api")
	}
	p.MetricsPath = "/api/metrics"
	p.Use(r)
}

func addDocs(r *gin.Engine) {
	docs.SwaggerInfo.BasePath = "/api"
	r.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
