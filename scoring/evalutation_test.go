package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandlePresence(t *testing.T) {
	objective := &repository.Objective{
		Id: 1,
		ScoringPreset: &repository.ScoringPreset{
			Points: repository.ExtendingNumberSlice{10},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	match1 := Match{
		TeamId:    1,
		Number:    1,
		UserId:    1,
		Finished:  false,
		Timestamp: now.Add(-24 * time.Hour),
	}
	match2 := Match{
		TeamId:    2,
		Number:    2,
		UserId:    2,
		Finished:  true,
		Timestamp: now.Add(-24 * time.Hour),
	}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2

	scores, err := handlePresence(objective, aggregations, []*Score{})
	assert.NoError(t, err)
	assert.Equal(t, 0, scores[0].Points)
	assert.Equal(t, 10, scores[1].Points)
}

func TestHandlePointsFromValue(t *testing.T) {
	value := 10.0
	objective := &repository.Objective{
		Id: 1,
		ScoringPreset: &repository.ScoringPreset{
			Points:   repository.ExtendingNumberSlice{value},
			PointCap: 500,
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	match1 := Match{
		TeamId:    1,
		Number:    1,
		UserId:    1,
		Finished:  false,
		Timestamp: now.Add(-24 * time.Hour),
	}
	match2 := Match{
		TeamId:    2,
		Number:    2,
		UserId:    2,
		Finished:  true,
		Timestamp: now.Add(-24 * time.Hour),
	}
	match3 := Match{
		TeamId:    3,
		Number:    100,
		UserId:    3,
		Finished:  true,
		Timestamp: now.Add(-24 * time.Hour),
	}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3

	scores, err := handlePointsFromValue(objective, aggregations, []*Score{})
	assert.NoError(t, err)
	assert.Equal(t, int(value*float64(match1.Number)), scores[0].Points)
	assert.Equal(t, int(value*float64(match2.Number)), scores[1].Points)
	assert.Equal(t, objective.ScoringPreset.PointCap, scores[2].Points)
}

func TestHandleRankedTime(t *testing.T) {
	objective := &repository.Objective{
		Id: 1,
		ScoringPreset: &repository.ScoringPreset{
			Points: repository.ExtendingNumberSlice{10, 5},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	match1 := Match{TeamId: 1, UserId: 1, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match2 := Match{TeamId: 2, UserId: 2, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match3 := Match{TeamId: 3, UserId: 3, Finished: true, Timestamp: now.Add(-23 * time.Hour)}
	match4 := Match{TeamId: 4, UserId: 4, Finished: true, Timestamp: now.Add(-22 * time.Hour)}
	match5 := Match{TeamId: 5, UserId: 5, Finished: false, Timestamp: now.Add(-21 * time.Hour)}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3
	aggregations[objective.Id][4] = &match4
	aggregations[objective.Id][5] = &match5

	scores, err := handleRankedTime(objective, aggregations, []*Score{})
	assert.NoError(t, err)
	assert.Equal(t, 1, scores[0].Rank)
	assert.Equal(t, 10, scores[0].Points)
	assert.Equal(t, 1, scores[1].Rank)
	assert.Equal(t, 10, scores[1].Points)
	assert.Equal(t, 2, scores[2].Rank)
	assert.Equal(t, 5, scores[2].Points)
	assert.Equal(t, 3, scores[3].Rank)
	assert.Equal(t, 5, scores[3].Points)
	assert.Equal(t, 0, scores[4].Rank)
	assert.Equal(t, 0, scores[4].Points)
}
func TestHandleRankedValue(t *testing.T) {
	objective := &repository.Objective{
		Id: 1,
		ScoringPreset: &repository.ScoringPreset{
			Points: repository.ExtendingNumberSlice{10, 5},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	match1 := Match{TeamId: 1, UserId: 1, Number: 4, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match2 := Match{TeamId: 2, UserId: 2, Number: 4, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match3 := Match{TeamId: 3, UserId: 3, Number: 3, Finished: true, Timestamp: now.Add(-23 * time.Hour)}
	match4 := Match{TeamId: 4, UserId: 4, Number: 2, Finished: true, Timestamp: now.Add(-22 * time.Hour)}
	match5 := Match{TeamId: 5, UserId: 5, Number: 1, Finished: false, Timestamp: now.Add(-21 * time.Hour)}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3
	aggregations[objective.Id][4] = &match4
	aggregations[objective.Id][5] = &match5

	scores, err := handleRankedValue(objective, aggregations, []*Score{})
	assert.NoError(t, err)
	assert.Equal(t, 1, scores[0].Rank)
	assert.Equal(t, 10, scores[0].Points)
	assert.Equal(t, 1, scores[1].Rank)
	assert.Equal(t, 10, scores[1].Points)
	assert.Equal(t, 2, scores[2].Rank)
	assert.Equal(t, 5, scores[2].Points)
	assert.Equal(t, 3, scores[3].Rank)
	assert.Equal(t, 5, scores[3].Points)
	assert.Equal(t, 0, scores[4].Rank)
	assert.Equal(t, 0, scores[4].Points)
}
func TestHandleRankedReverse(t *testing.T) {
	objective := &repository.Objective{
		Id: 1,
		ScoringPreset: &repository.ScoringPreset{
			Points: repository.ExtendingNumberSlice{10, 5},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	match1 := Match{TeamId: 1, UserId: 1, Number: 1, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match2 := Match{TeamId: 2, UserId: 2, Number: 1, Finished: true, Timestamp: now.Add(-24 * time.Hour)}
	match3 := Match{TeamId: 3, UserId: 3, Number: 2, Finished: true, Timestamp: now.Add(-23 * time.Hour)}
	match4 := Match{TeamId: 4, UserId: 4, Number: 3, Finished: true, Timestamp: now.Add(-22 * time.Hour)}
	match5 := Match{TeamId: 5, UserId: 5, Number: 4, Finished: false, Timestamp: now.Add(-21 * time.Hour)}

	aggregations[objective.Id][1] = &match1
	aggregations[objective.Id][2] = &match2
	aggregations[objective.Id][3] = &match3
	aggregations[objective.Id][4] = &match4
	aggregations[objective.Id][5] = &match5

	scores, err := handleRankedReverse(objective, aggregations, []*Score{})
	assert.NoError(t, err)
	assert.Equal(t, 1, scores[0].Rank)
	assert.Equal(t, 10, scores[0].Points)
	assert.Equal(t, 1, scores[1].Rank)
	assert.Equal(t, 10, scores[1].Points)
	assert.Equal(t, 2, scores[2].Rank)
	assert.Equal(t, 5, scores[2].Points)
	assert.Equal(t, 3, scores[3].Rank)
	assert.Equal(t, 5, scores[3].Points)
	assert.Equal(t, 0, scores[4].Rank)
	assert.Equal(t, 0, scores[4].Points)
}

func TestHandleChildBonus(t *testing.T) {
	objective := &repository.Objective{
		Id: 10,
		ScoringPreset: &repository.ScoringPreset{
			Points: repository.ExtendingNumberSlice{10, 9, 5},
		},
		Children: utils.Map([]int{1, 2, 3, 4, 5}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()
	childScores := []*Score{
		{Id: 1, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-24 * time.Hour)},
		{Id: 2, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-23 * time.Hour)},
		{Id: 3, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-22 * time.Hour)},
		{Id: 4, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-21 * time.Hour)},
		{Id: 5, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-20 * time.Hour)},
		{Id: 1, TeamId: 2, Points: 10, Finished: true, Timestamp: now.Add(-20 * time.Hour)},
	}

	scores, err := handleChildBonus(objective, make(ObjectiveTeamMatches), childScores)
	assert.NoError(t, err)
	assert.Equal(t, 0, scores[0].Points)

	idTeamIdScoreMap := make(map[int]map[int]*Score)
	for _, score := range childScores {
		if _, exists := idTeamIdScoreMap[score.Id]; !exists {
			idTeamIdScoreMap[score.Id] = make(map[int]*Score)
		}
		idTeamIdScoreMap[score.Id][score.TeamId] = score
	}
	assert.Equal(t, 20, idTeamIdScoreMap[1][1].Points)
	assert.Equal(t, 19, idTeamIdScoreMap[2][1].Points)
	assert.Equal(t, 15, idTeamIdScoreMap[3][1].Points)
	assert.Equal(t, 15, idTeamIdScoreMap[4][1].Points)
	assert.Equal(t, 15, idTeamIdScoreMap[5][1].Points)
	assert.Equal(t, 20, idTeamIdScoreMap[1][2].Points)
}

func TestHandleChildRanking(t *testing.T) {
	objective := &repository.Objective{
		Id: 10,
		ScoringPreset: &repository.ScoringPreset{
			Points: repository.ExtendingNumberSlice{20, 10},
		},
		Children: utils.Map([]int{1, 2}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()
	childScores := []*Score{
		{Id: 1, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-23 * time.Hour)},
		{Id: 2, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-23 * time.Hour)},
		{Id: 1, TeamId: 2, Points: 10, Finished: true, Timestamp: now.Add(-22 * time.Hour)},
		{Id: 2, TeamId: 2, Points: 10, Finished: true, Timestamp: now.Add(-24 * time.Hour)},
		{Id: 1, TeamId: 3, Points: 10, Finished: true, Timestamp: now.Add(-20 * time.Hour)},
	}

	scores, err := handleChildRanking(objective, make(ObjectiveTeamMatches), childScores)
	assert.NoError(t, err)
	idTeamIdScoreMap := make(map[int]map[int]*Score)
	for _, score := range scores {
		if _, exists := idTeamIdScoreMap[score.Id]; !exists {
			idTeamIdScoreMap[score.Id] = make(map[int]*Score)
		}
		idTeamIdScoreMap[score.Id][score.TeamId] = score
	}
	assert.Equal(t, 20, idTeamIdScoreMap[10][1].Points)
	assert.Equal(t, 10, idTeamIdScoreMap[10][2].Points)
	assert.Equal(t, 0, idTeamIdScoreMap[10][3].Points)

}

func TestHandleBingo(t *testing.T) {
	objective := &repository.Objective{
		Id: 10,
		ScoringPreset: &repository.ScoringPreset{
			Points: repository.ExtendingNumberSlice{30, 20, 10},
		},
		Children: utils.Map([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()
	childScores := []*Score{
		{Id: 1, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-24 * time.Hour)},
		{Id: 2, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-23 * time.Hour)},
		{Id: 3, TeamId: 1, Points: 10, Finished: true, Timestamp: now.Add(-22 * time.Hour)},
		{Id: 4, TeamId: 2, Points: 10, Finished: true, Timestamp: now.Add(-24 * time.Hour)},
		{Id: 5, TeamId: 2, Points: 10, Finished: true, Timestamp: now.Add(-22 * time.Hour)},
		{Id: 6, TeamId: 3, Points: 10, Finished: true, Timestamp: now.Add(-22 * time.Hour)},
	}

	scores, err := handleBingoN(2)(objective, make(ObjectiveTeamMatches), childScores)
	assert.NoError(t, err)
	idTeamIdScoreMap := make(map[int]map[int]*Score)
	for _, score := range scores {
		if _, exists := idTeamIdScoreMap[score.Id]; !exists {
			idTeamIdScoreMap[score.Id] = make(map[int]*Score)
		}
		idTeamIdScoreMap[score.Id][score.TeamId] = score
	}
	for _, score := range childScores {
		if _, exists := idTeamIdScoreMap[score.Id]; !exists {
			idTeamIdScoreMap[score.Id] = make(map[int]*Score)
		}
		idTeamIdScoreMap[score.Id][score.TeamId] = score
	}
	assert.Equal(t, 30, idTeamIdScoreMap[10][1].Points)
	assert.Equal(t, 20, idTeamIdScoreMap[10][2].Points)

	assert.Equal(t, 10, idTeamIdScoreMap[1][1].Points)
	assert.Equal(t, 10, idTeamIdScoreMap[2][1].Points)
	assert.Equal(t, 0, idTeamIdScoreMap[3][1].Points)
	assert.Equal(t, 10, idTeamIdScoreMap[4][2].Points)
	assert.Equal(t, 10, idTeamIdScoreMap[5][2].Points)
	assert.Equal(t, 10, idTeamIdScoreMap[6][3].Points)

}
