package main

import (
	"bpl/config"
	"bpl/controller"
	"log"

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
	controller.SetRoutes(r, db)

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
