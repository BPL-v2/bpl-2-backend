package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type ObjectiveIdTeamId struct {
	ObjectiveID int
	TeamID      int
}

type FreshMatches map[ObjectiveIdTeamId]bool

func (f FreshMatches) contains(match *Match) bool {
	return f[ObjectiveIdTeamId{ObjectiveID: match.ObjectiveID, TeamID: match.TeamID}]
}

type Match struct {
	ObjectiveID int
	Number      int
	Timestamp   time.Time
	UserID      int
	TeamID      int
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

func AggregateMatches(db *gorm.DB, event *repository.Event, objectives []*repository.Objective) (ObjectiveTeamMatches, error) {
	aggregations := make(ObjectiveTeamMatches)
	teamIds := utils.Map(event.Teams, func(team *repository.Team) int {
		return team.ID
	})
	objectiveMap := make(map[int]repository.Objective)
	objectiveIdLists := make(map[repository.AggregationType][]int)
	for _, objective := range objectives {
		objectiveIdLists[objective.Aggregation] = append(objectiveIdLists[objective.Aggregation], objective.ID)
		objectiveMap[objective.ID] = *objective
		aggregations[objective.ID] = make(TeamMatches)
	}
	wg := sync.WaitGroup{}
	for _, aggregation := range []repository.AggregationType{
		repository.EARLIEST_FRESH_ITEM,
		repository.EARLIEST,
		repository.MAXIMUM,
		repository.MINIMUM,
		repository.SUM_LATEST,
	} {
		wg.Add(1)
		go func(aggregation repository.AggregationType) {
			defer wg.Done()
			matches, err := aggregationMap[aggregation](db, objectiveIdLists[aggregation], teamIds, event.ID)
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, match := range matches {
				match.Finished = objectiveMap[match.ObjectiveID].RequiredAmount <= match.Number
				aggregations[match.ObjectiveID][match.TeamID] = match
			}
		}(aggregation)
	}
	wg.Wait()
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
	var wg sync.WaitGroup
	var freshMatches FreshMatches
	var firstMatches []*Match
	var err1, err2 error

	wg.Add(2)

	go func() {
		defer wg.Done()
		t := time.Now()
		freshMatches, err1 = getFreshMatches(db, objectiveIds, teamIds, eventId)
		fmt.Println("Fresh matches took", time.Since(t))
	}()

	go func() {
		defer wg.Done()
		t := time.Now()
		firstMatches, err2 = handleEarliest(db, objectiveIds, teamIds, eventId)
		fmt.Println("Earliest matches took", time.Since(t))
	}()

	wg.Wait()

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
            stash_id, 
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
