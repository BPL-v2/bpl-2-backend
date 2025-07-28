package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"log"
	"sort"
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

type AggregationHandler func(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error)

var aggregationMap = map[repository.AggregationType]AggregationHandler{
	repository.AggregationTypeEarliestFreshItem: handleEarliestFreshItem,
	repository.AggregationTypeEarliest:          handleEarliest,
	repository.AggregationTypeSumLatest:         handleLatestSum,
	repository.AggregationTypeMaximum:           handleMaximum,
	repository.AggregationTypeMinimum:           handleMinimum,
	repository.AggregationTypeDifferenceBetween: handleDifferenceBetween,
}
var scoreAggregationDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "score_aggregation_duration_s",
	Help: "Duration of Aggregation step during scoring",
}, []string{"aggregation-step"})

func AggregateMatches(db *gorm.DB, event *repository.Event, objectives []*repository.Objective) (ObjectiveTeamMatches, error) {
	totalTime := time.Now()
	aggregations := make(ObjectiveTeamMatches)
	teamIds := utils.Map(event.Teams, func(team *repository.Team) int {
		return team.Id
	})
	objectiveMap := make(map[int]repository.Objective)
	objectivesByAggregation := make(map[repository.AggregationType][]*repository.Objective)
	for _, objective := range objectives {
		objectivesByAggregation[objective.Aggregation] = append(objectivesByAggregation[objective.Aggregation], objective)
		objectiveMap[objective.Id] = *objective
		aggregations[objective.Id] = make(TeamMatches)
	}
	for _, aggregation := range []repository.AggregationType{
		repository.AggregationTypeEarliestFreshItem,
		repository.AggregationTypeEarliest,
		repository.AggregationTypeMaximum,
		repository.AggregationTypeMinimum,
		repository.AggregationTypeSumLatest,
		repository.AggregationTypeDifferenceBetween,
	} {
		if handler, ok := aggregationMap[aggregation]; ok {
			t := time.Now()
			matches, err := handler(db, objectivesByAggregation[aggregation], teamIds, event.Id)
			if err != nil {
				log.Print(err)
				continue
			}
			for _, match := range matches {
				// todo: maybe move this into the aggregation steps
				if aggregation != repository.AggregationTypeDifferenceBetween {
					match.Finished = objectiveMap[match.ObjectiveId].RequiredAmount <= match.Number
				}
				aggregations[match.ObjectiveId][match.TeamId] = match
			}
			scoreAggregationDuration.WithLabelValues(string(aggregation)).Set(time.Since(t).Seconds())
		}
	}
	scoreAggregationDuration.WithLabelValues("total").Set(time.Since(totalTime).Seconds())
	return aggregations, nil
}

func getObjectiveIds(objectives []*repository.Objective) []int {
	return utils.Map(objectives, func(objective *repository.Objective) int {
		return objective.Id
	})
}

func handleEarliest(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
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
					match.timestamp ASC
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
	err := db.Raw(query, map[string]interface{}{"objectiveIds": getObjectiveIds(objectives), "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func handleEarliestFreshItem(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	freshMatches, err := getFreshMatches(db, objectives, teamIds, eventId)
	if err != nil {
		return nil, err
	}
	firstMatches, err := handleEarliest(db, objectives, teamIds, eventId)
	if err != nil {
		return nil, err
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
	switch aggregationType {
	case repository.AggregationTypeMaximum:
		operator = "MAX"
	case repository.AggregationTypeMinimum:
		operator = "MIN"
	default:
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

func handleMaximum(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	t := time.Now()
	query, err := getExtremeQuery(repository.AggregationTypeMaximum)
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	err = db.Raw(query, map[string]interface{}{"objectiveIds": getObjectiveIds(objectives), "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	scoreAggregationDuration.WithLabelValues("handleMaximum").Set(time.Since(t).Seconds())
	return matches, nil
}

func handleMinimum(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	query, err := getExtremeQuery(repository.AggregationTypeMinimum)
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	err = db.Raw(query,
		map[string]interface{}{"objectiveIds": getObjectiveIds(objectives), "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func handleLatestSum(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
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
	err := db.Raw(query, map[string]interface{}{"objectiveIds": getObjectiveIds(objectives), "teamIds": teamIds, "eventId": eventId}).Scan(&matches).Error
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func getFreshMatches(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) (FreshMatches, error) {
	// todo: might want to also check if the match finishes the objective
	t := time.Now()
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
	result := db.Raw(query, map[string]interface{}{"objectiveIds": getObjectiveIds(objectives), "teamIds": teamIds, "eventId": eventId}).Scan(&matchList)
	if result.Error != nil {
		return nil, result.Error
	}
	freshMatches := make(FreshMatches)
	for _, id := range matchList {
		freshMatches[id] = true
	}
	scoreAggregationDuration.WithLabelValues("getFreshMatches").Set(time.Since(t).Seconds())
	return freshMatches, nil
}

func handleDifferenceBetween(db *gorm.DB, objectives []*repository.Objective, teamIds []int, eventId int) ([]*Match, error) {
	query := `
	SELECT
		match.objective_id,
		team_users.team_id,
		match.user_id,
		match.number,
		match.timestamp
	FROM
		objective_matches AS match
	JOIN
		team_users ON team_users.user_id = match.user_id
	WHERE
		match.event_id = @eventId AND match.objective_id IN @objectiveIds AND team_users.team_id IN @teamIds
	ORDER BY
		match.objective_id, match.timestamp
	`
	objectiveMap := make(map[int]repository.Objective)
	for _, objective := range objectives {
		objectiveMap[objective.Id] = *objective
	}
	preMatches := make([]*Match, 0)
	err := db.Raw(query, map[string]interface{}{"objectiveIds": getObjectiveIds(objectives), "teamIds": teamIds, "eventId": eventId}).Scan(&preMatches).Error
	if err != nil {
		return nil, err
	}
	matches := make([]*Match, 0)
	for _, objective := range objectives {
		if objective.ValidFrom == nil || objective.ValidTo == nil {
			fmt.Printf("DIFFERENCE_BETWEEN objective %d does not have timestamps set\n", objective.Id)
			continue
		}

		matches = append(matches, getDifferencesBetweenTimestamps(objective, preMatches, teamIds)...)
	}
	return matches, nil

}

func getDifferencesBetweenTimestamps(objective *repository.Objective, preMatches []*Match, teamIds []int) []*Match {
	matches := []*Match{}
	for _, teamId := range teamIds {
		objectiveMatches := utils.Filter(preMatches, func(match *Match) bool {
			return match.ObjectiveId == objective.Id && match.TeamId == teamId
		})
		sort.Slice(objectiveMatches, func(i, j int) bool {
			return objectiveMatches[i].Timestamp.Before(objectiveMatches[j].Timestamp)
		})
		if len(objectiveMatches) == 0 {
			continue
		}
		minMatch := &Match{
			Timestamp: objectiveMatches[0].Timestamp.Add(-time.Hour),
			Number:    0,
		}
		maxMatch := objectiveMatches[0]
		for _, match := range objectiveMatches {
			if match.Timestamp.Before(*objective.ValidFrom) && minMatch.Timestamp.Before(match.Timestamp) {
				minMatch = match
			}
			if match.Timestamp.Before(*objective.ValidTo) && maxMatch.Timestamp.Before(match.Timestamp) {
				maxMatch = match
			}
		}
		matches = append(matches, &Match{
			ObjectiveId: objective.Id,
			Number:      maxMatch.Number - minMatch.Number,
			Timestamp:   maxMatch.Timestamp,
			UserId:      0,
			TeamId:      maxMatch.TeamId,
			Finished:    time.Now().After(*objective.ValidTo),
		})
	}
	return matches
}
