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

	ginprometheus "github.com/zsais/go-gin-prometheus"

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
	_, err = config.InitDB(
		os.Getenv("DATABASE_HOST"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("DATABASE_NAME"),
	)
	if err != nil {
		panic(err)
	}
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
	// )

	// if err != nil {
	// 	panic(err)
	// }

	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},                                // Allow all origins
		AllowMethods:     []string{"GET", "OPTIONS"},                   // Allow only GET requests
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"}, // Allow necessary headers
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false, // Credentials are not allowed when AllowOrigins is "*"
		MaxAge:           12 * time.Hour,
	}))
	addMetrics(r)
	addDocs(r)

	controller.SetRoutes(r)
	r.Run(":8000")
}

func addMetrics(r *gin.Engine) {
	p := ginprometheus.NewPrometheus("gin")
	re := regexp.MustCompile(`\d+`)
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := re.ReplaceAllString(c.Request.URL.String(), "?")
		return strings.TrimPrefix(url, "/api")
	}
	p.MetricsPath = "/api/metrics"
	p.Use(r)
}

func addDocs(r *gin.Engine) {
	docs.SwaggerInfo.BasePath = "/api"
	r.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
