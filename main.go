package main

import (
	"bpl/config"
	"bpl/controller"
	"bpl/docs"
	"bpl/repository"
	"fmt"
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
	"gorm.io/gorm"
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
	t := time.Now()
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
	// autoMigrate(db)

	r := gin.Default()
	r.SetTrustedProxies(nil)
	addMetrics(r)
	addDocs(r)
	setCors(r)

	controller.SetRoutes(r)
	fmt.Println("Server started in", time.Since(t))
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

func setCors(r *gin.Engine) {
	corsConfigGetOptions := cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	corsConfigOtherMethods := cors.Config{
		AllowOrigins: []string{
			"https://bpl-poe.com",
			"https://bpl-2.netlify.app",
			"https://v2202503259898322516.goodsrv.de",
			"http://localhost",
			"http://localhost:3000",
		},
		AllowMethods:     []string{"POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	r.Use(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			// Check the Access-Control-Request-Method header to determine the actual method being preflighted
			requestedMethod := c.GetHeader("Access-Control-Request-Method")
			if requestedMethod == "GET" || requestedMethod == "OPTIONS" {
				cors.New(corsConfigGetOptions)(c)
			} else {
				cors.New(corsConfigOtherMethods)(c)
			}
			c.AbortWithStatus(204) // Respond with 204 No Content for preflight
			return
		}

		if c.Request.Method == "GET" {
			cors.New(corsConfigGetOptions)(c)
		} else {
			cors.New(corsConfigOtherMethods)(c)
		}
	})
}

func autoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
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
		&repository.RecurringJob{},
		&repository.LadderEntry{},
	)
	if err != nil {
		panic(err)
	}
}
