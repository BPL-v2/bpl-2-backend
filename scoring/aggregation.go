package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

type ObjectiveIdTeamId struct {
	ObjectiveId int
	TeamId      int
}

type FreshMatches map[ObjectiveIdTeamId]bool

func (f FreshMatches) contains(match *Match) bool {
	return f[ObjectiveIdTeamId{ObjectiveId: match.ObjectiveId, TeamId: match.TeamId}]
}

type Match struct {
	ObjectiveId int
	Number      int
	Timestamp   time.Time
	UserId      int
	TeamId      int
	Finished    bool
}

type TeamMatches = map[int]*Match

type ObjectiveTeamMatches = map[int]TeamMatches

var aggregationMap = map[repository.AggregationType]func(db *gorm.DB, teamIds []int, objectiveIds []int, eventId int) ([]*Match, error){
	repository.EARLIEST_FRESH_ITEM: handleEarliestFreshItem,
	repository.EARLIEST:            handleEarliest,
	repository.SUM_LATEST:          handleLatestSum,
	repository.MAXIMUM:             handleMaximum,
	repository.MINIMUM:             handleMinimum,
}
var scoreAggregationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "score_aggregation_duration_s",
	Help: "Duration of Aggregation step during scoring",
	Buckets: []float64{
		0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 60,
	},
})

func AggregateMatches(db *gorm.DB, event *repository.Event, objectives []*repository.Objective) (ObjectiveTeamMatches, error) {
	timer := prometheus.NewTimer(scoreAggregationDuration)
	defer timer.ObserveDuration()
	aggregations := make(ObjectiveTeamMatches)
	teamIds := utils.Map(event.Teams, func(team *repository.Team) int {
		return team.Id
	})
	objectiveMap := make(map[int]repository.Objective)
	objectiveIdLists := make(map[repository.AggregationType][]int)
	for _, objective := range objectives {
		objectiveIdLists[objective.Aggregation] = append(objectiveIdLists[objective.Aggregation], objective.Id)
		objectiveMap[objective.Id] = *objective
		aggregations[objective.Id] = make(TeamMatches)
	}
	// wg := sync.WaitGroup{}
	for _, aggregation := range []repository.AggregationType{
		repository.EARLIEST_FRESH_ITEM,
		repository.EARLIEST,
		repository.MAXIMUM,
		repository.MINIMUM,
		repository.SUM_LATEST,
	} {
		// wg.Add(1)
		// go func(aggregation repository.AggregationType) {
		// 	defer wg.Done()
		matches, err := aggregationMap[aggregation](db, objectiveIdLists[aggregation], teamIds, event.Id)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		for _, match := range matches {
			match.Finished = objectiveMap[match.ObjectiveId].RequiredAmount <= match.Number
			aggregations[match.ObjectiveId][match.TeamId] = match
		}
		// }(aggregation)
	}
	// wg.Wait()
	return aggregations, nil
}

func handleEarliest(db *gorm.DB, objectiveIds []int, teamIds []int, eventId int) ([]*Match, error) {
	query := `
	WITH ranked_matches AS (
		SELECT 
			match.objective_id,
			match.number,
			match.timestamp,
			match.user_id, 
			match.number >= objectives.required_amount AS finished,
			RANK() OVER (
				PARTITION BY match.objective_id, team_users.team_id
				ORDER BY
					CASE 
						WHEN match.number >= objectives.required_amount THEN 1000000
						ELSE match.number
					END DESC,
					match.timestamp ASC,
					match.id ASC
			) AS rank,
			team_users.team_id
		FROM 
			objective_matches as match
		JOIN 
			objectives ON objectives.id = match.objective_id
		JOIN 
			team_users ON team_users.user_id = match.user_id
		WHERE 
			match.event_id = @eventId AND match.objective_id IN @objectiveIds AND team_users.team_id IN @teamIds
	)
	SELECT 
		*
	FROM 
		ranked_matches
	WHERE 
		rank = 1;
	`

	matches := make([]*Match, 0)
	err := db.Raw(query, map[string]interface{}{"objectiveIds": objectiveIds, "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func handleEarliestFreshItem(db *gorm.DB, objectiveIds []int, teamIds []int, eventId int) ([]*Match, error) {
	// var wg sync.WaitGroup
	var freshMatches FreshMatches
	var firstMatches []*Match
	var err1, err2 error

	// wg.Add(2)

	// go func() {
	// 	defer wg.Done()
	freshMatches, err1 = getFreshMatches(db, objectiveIds, teamIds, eventId)
	// }()

	// go func() {
	// 	defer wg.Done()
	firstMatches, err2 = handleEarliest(db, objectiveIds, teamIds, eventId)
	// }()

	// wg.Wait()

	if err1 != nil {
		return nil, err1
	}
	if err2 != nil {
		return nil, err2
	}
	matches := make([]*Match, 0)
	for _, match := range firstMatches {
		if freshMatches.contains(match) {
			matches = append(matches, match)
		}
	}
	return matches, nil
}

func getExtremeQuery(aggregationType repository.AggregationType) (string, error) {
	var operator string
	if aggregationType == repository.MAXIMUM {
		operator = "MAX"
	} else if aggregationType == repository.MINIMUM {
		operator = "MIN"
	} else {
		return "", fmt.Errorf("invalid aggregation type")
	}
	return fmt.Sprintf(`
    WITH extreme AS (
        SELECT
            match.objective_id,
            team_users.team_id,
            %s(match.number) AS number
        FROM
            objective_matches AS match
        JOIN
            team_users ON team_users.user_id = match.user_id
        WHERE
			match.event_id = @eventId AND match.objective_id IN @objectiveIds AND team_users.team_id IN @teamIds
        GROUP BY
            match.objective_id, team_users.team_id
    )
    SELECT
        extreme.objective_id,
        extreme.team_id,
        match.user_id,
        extreme.number,
		match.timestamp
    FROM
        extreme
    JOIN
        objective_matches AS match ON match.objective_id = extreme.objective_id
		AND match.event_id = @eventId
        AND match.number = extreme.number
        AND match.user_id IN (
            SELECT user_id
            FROM team_users
            WHERE team_users.team_id = extreme.team_id
        )
 	`, operator), nil

}

func handleMaximum(db *gorm.DB, objectiveIds []int, teamIds []int, eventId int) ([]*Match, error) {
	query, err := getExtremeQuery(repository.MAXIMUM)
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	err = db.Raw(query, map[string]interface{}{"objectiveIds": objectiveIds, "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func handleMinimum(db *gorm.DB, objectiveIds []int, teamIds []int, eventId int) ([]*Match, error) {
	query, err := getExtremeQuery(repository.MINIMUM)
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	err = db.Raw(query,
		map[string]interface{}{"objectiveIds": objectiveIds, "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func handleLatestSum(db *gorm.DB, objectiveIds []int, teamIds []int, eventId int) ([]*Match, error) {
	query := `
	WITH latest AS (
		SELECT
			match.objective_id,
			match.user_id,
			MAX(timestamp) AS timestamp
		FROM
			objective_matches AS match
		WHERE
			match.objective_id IN @objectiveIds AND match.event_id = @eventId
		GROUP BY
			match.objective_id, match.user_id 
	)		
	SELECT
		match.objective_id,
		team_users.team_id,
		SUM(match.number) AS number,
        MAX(match.timestamp) AS timestamp
	FROM
		objective_matches AS match
	JOIN
		latest ON latest.objective_id = match.objective_id
		AND latest.user_id = match.user_id
	JOIN
		team_users ON team_users.user_id = match.user_id
	WHERE
		team_users.team_id IN @teamIds AND match.event_id = @eventId
	GROUP BY
		match.objective_id, team_users.team_id
	`
	matches := make([]*Match, 0)
	err := db.Raw(query, map[string]interface{}{"objectiveIds": objectiveIds, "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func getFreshMatches(db *gorm.DB, objectiveIds []int, teamIds []int, eventId int) (FreshMatches, error) {
	// todo: might want to also check if the match finishes the objective
	query := `
    WITH latest AS (
        SELECT 
            MAX(id) AS id
        FROM stash_changes
		WHERE event_id = @eventId
        GROUP BY stash_id
    )
    SELECT 
        objective_matches.objective_id,
        team_users.team_id
    FROM objective_matches
	JOIN latest ON objective_matches.stash_change_id = latest.id
    JOIN team_users ON team_users.user_id = objective_matches.user_id
    WHERE event_id = @eventId AND objective_matches.objective_id IN @objectiveIds AND team_users.team_id IN @teamIds
    GROUP BY 
        objective_matches.objective_id,
        team_users.team_id
    `
	matchList := make([]ObjectiveIdTeamId, 0)
	result := db.Raw(query, map[string]interface{}{"objectiveIds": objectiveIds, "teamIds": teamIds, "eventId": eventId}).Scan(&matchList)
	if result.Error != nil {
		return nil, result.Error
	}
	freshMatches := make(FreshMatches)
	for _, id := range matchList {
		freshMatches[id] = true
	}

	return freshMatches, nil
}
