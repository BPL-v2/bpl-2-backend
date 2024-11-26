package main

import (
	"bpl/config"
	"bpl/controller"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	db, err := config.InitDB()
	if err != nil {
		log.Fatal(err)
		return
	}
	_ = db
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	controller.SetRoutes(r, db)
	// conf := &oauth2.Config{
	// 	ClientID:     "YOUR_CLIENT_ID",
	// 	ClientSecret: "YOUR_CLIENT_SECRET",
	// 	Scopes:       []string{"SCOPE1", "SCOPE2"},
	// 	Endpoint: oauth2.Endpoint{
	// 		AuthURL:  "https://provider.com/o/oauth2/auth",
	// 		TokenURL: "https://provider.com/o/oauth2/token",
	// 	},
	// }

	// use PKCE to protect against CSRF attacks
	// https://www.ietf.org/archive/id/draft-ietf-oauth-security-topics-22.html#name-countermeasures-6
	// verifier := oauth2.GenerateVerifier()

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	// url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	// fmt.Printf("Visit the URL for the auth dialog: %v", url)

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	// var code string
	// if _, err := fmt.Scan(&code); err != nil {
	// 	log.Fatal(err)
	// }
	// tok, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	r.Run(":8000") // listen and serve on 0.0.0.0:8080

}

// func createObjective(db *gorm.DB, objectiveName string, conditions []model.Condition) error {
// 	objective := model.Objective{Name: objectiveName, RequiredNumber: "1"}
// 	if err := db.Create(&objective).Error; err != nil {
// 		return err
// 	}

// 	for i := range conditions {
// 		conditions[i].ObjectiveID = objective.ID
// 	}

// 	if err := db.Create(&conditions).Error; err != nil {
// 		return err
// 	}
// 	return nil
// }

// func getAllObjectives(db *gorm.DB) ([]*model.Objective, error) {
// 	return getObjectives(db, "")
// }

// func getObjectiveById(db *gorm.DB, id int) (*model.Objective, error) {
// 	objs, err := getObjectives(db, "objectives.id = ? and objectives.name = ?", id, "objective1")
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(objs) == 0 {
// 		return nil, fmt.Errorf("objective with id %d not found", id)
// 	}
// 	return objs[0], nil
// }

// func getObjectives(db *gorm.DB, condition string, args ...interface{}) ([]*model.Objective, error) {
// 	var objectives []*model.Objective

// 	// Perform a query to get objectives and their conditions using Preload
// 	query := db.Preload("Conditions")

// 	if condition != "" {
// 		query = query.Where(condition, args...)
// 	}
// 	if err := query.Find(&objectives).Error; err != nil {
// 		return nil, err
// 	}

// 	return objectives, nil
// }
