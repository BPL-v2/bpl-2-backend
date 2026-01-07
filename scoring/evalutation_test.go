package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandlePresence(t *testing.T) {
	// This tests PRESENCE scoring where teams get points simply for completing an objective
	// Only teams with Finished=true should receive points
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10},
			},
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

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handlePresence(objective, objective.ScoringPresets[0], aggregations, scoreMap)
	assert.NoError(t, err, "handlePresence should not return an error")

	// Verify only the finished team gets points
	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 (not finished) should have 0 points")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 (finished) should have 10 points")
}

func TestHandlePointsFromValue(t *testing.T) {
	// This tests POINTS_FROM_VALUE scoring where points are calculated by multiplying
	// the match Number (e.g., item count, completion %) by a point value
	// Also tests that PointCap limits the maximum points
	value := 10.0
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:       presetId,
				Points:   repository.ExtendingNumberSlice{value},
				PointCap: 500,
			},
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

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handlePointsFromValue(objective, objective.ScoringPresets[0], aggregations, scoreMap)
	assert.NoError(t, err, "handlePointsFromValue should not return an error")

	// Verify points are calculated correctly and capped when necessary
	assert.Equal(t, int(value*float64(match1.Number)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 points should be value * Number (10 * 1 = 10)")
	assert.Equal(t, int(value*float64(match2.Number)), scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 points should be value * Number (10 * 2 = 20)")
	assert.Equal(t, objective.ScoringPresets[0].PointCap, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 points should be capped at PointCap (500)")
}

func TestHandleRankedTime(t *testing.T) {
	// This tests RANKED_TIME scoring where teams are ranked by completion time
	// Earlier completion = better rank. Ties result in same rank. Unfinished teams get 0 points.
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 5},
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	// Teams 1 & 2: Finished at -24h (tied) -> rank 1, 10 points each
	// Team 3: Finished at -23h -> rank 2, 5 points
	// Team 4: Finished at -22h -> rank 3, 5 points (extends from last)
	// Team 5: Not finished -> rank 0, 0 points
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

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleRankedTime(objective, objective.ScoringPresets[0], aggregations, scoreMap)
	assert.NoError(t, err, "handleRankedTime should not return an error")

	// Verify ranks and points are assigned correctly based on completion time
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (finished at -24h, tied)")
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 10 points (rank 1)")
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 1 (finished at -24h, tied)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (rank 1)")
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 2 (finished at -23h)")
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 5 points (rank 2)")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (finished at -22h)")
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 5 points (rank 3, extends)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 0 (not finished)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 0 points (not finished)")
}
func TestHandleRankedValue(t *testing.T) {
	// This tests RANKED_VALUE scoring where teams are ranked by their Number value (higher = better)
	// Used for objectives where a higher count/value is better (e.g., boss kills, items collected)
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 5},
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	// Teams 1 & 2: Number=4 (tied) -> rank 1, 10 points each
	// Team 3: Number=3 -> rank 2, 5 points
	// Team 4: Number=2 -> rank 3, 5 points
	// Team 5: Not finished -> rank 0, 0 points
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

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleRankedValue(objective, objective.ScoringPresets[0], aggregations, scoreMap)
	assert.NoError(t, err, "handleRankedValue should not return an error")

	// Verify ranks and points are assigned correctly based on Number value
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (Number=4, tied)")
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 10 points (rank 1)")
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 1 (Number=4, tied)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (rank 1)")
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 2 (Number=3)")
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 5 points (rank 2)")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (Number=2)")
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 5 points (rank 3, extends)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 0 (not finished)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 0 points (not finished)")
}
func TestHandleRankedReverse(t *testing.T) {
	// This tests RANKED_REVERSE scoring where teams are ranked by Number value (lower = better)
	// Used for objectives where a lower value is better (e.g., fastest time, fewest deaths)
	presetId := 100
	objective := &repository.Objective{
		Id: 1,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 5},
			},
		},
	}
	aggregations := make(ObjectiveTeamMatches)
	aggregations[objective.Id] = make(TeamMatches)
	now := time.Now()
	// Teams 1 & 2: Number=1 (lowest, tied) -> rank 1, 10 points each
	// Team 3: Number=2 -> rank 2, 5 points
	// Team 4: Number=3 -> rank 3, 5 points
	// Team 5: Not finished -> rank 0, 0 points
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

	scoreMap := make(map[int]map[int]*Score)
	for teamId := range aggregations[objective.Id] {
		scoreMap[teamId] = make(map[int]*Score)
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleRankedReverse(objective, objective.ScoringPresets[0], aggregations, scoreMap)
	assert.NoError(t, err, "handleRankedReverse should not return an error")

	// Verify ranks and points - lower Number values get better ranks
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (Number=1, lowest, tied)")
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 10 points (rank 1)")
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 1 (Number=1, lowest, tied)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (rank 1)")
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 2 (Number=2)")
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 5 points (rank 2)")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (Number=3)")
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 5 points (rank 3, extends)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 0 (not finished)")
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 0 points (not finished)")
}

func TestHandleChildBonus(t *testing.T) {
	// This tests BONUS_PER_COMPLETION scoring where bonus points are awarded to child objectives
	// based on completion order. Earlier completions get higher bonuses.
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{10, 9, 5},
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4, 5}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()

	// Build scoreMap with parent and child scores
	// Team 1: Completes 5 children -> bonuses: 10, 9, 5, 5, 5 (by completion order)
	// Team 2: Completes 1 child -> bonus: 10
	scoreMap := make(map[int]map[int]*Score)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-23 * time.Hour), true},
		{3, 1, now.Add(-22 * time.Hour), true},
		{4, 1, now.Add(-21 * time.Hour), true},
		{5, 1, now.Add(-20 * time.Hour), true},
		{1, 2, now.Add(-20 * time.Hour), true},
	}

	// Initialize all teams with all child objectives (even if not finished)
	for teamId := 1; teamId <= 2; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 5; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Now update with actual finished data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Add parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildBonus(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildBonus should not return an error")

	// Verify BonusPoints were added to child scores in order of completion
	assert.Equal(t, 10, scoreMap[1][1].BonusPoints, "Team 1, child 1 should have 10 bonus (1st completed)")
	assert.Equal(t, 9, scoreMap[1][2].BonusPoints, "Team 1, child 2 should have 9 bonus (2nd completed)")
	assert.Equal(t, 5, scoreMap[1][3].BonusPoints, "Team 1, child 3 should have 5 bonus (3rd completed)")
	assert.Equal(t, 5, scoreMap[1][4].BonusPoints, "Team 1, child 4 should have 5 bonus (4th completed, extends)")
	assert.Equal(t, 5, scoreMap[1][5].BonusPoints, "Team 1, child 5 should have 5 bonus (5th completed, extends)")
	assert.Equal(t, 10, scoreMap[2][1].BonusPoints, "Team 2, child 1 should have 10 bonus (1st completed)")
}

func TestHandleChildRanking(t *testing.T) {
	// This tests RANKED_COMPLETION scoring where teams are ranked by completing ALL child objectives
	// Only teams that complete all children get points. Ranking is by completion time of last child.
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{20, 10},
			},
		},
		Children: utils.Map([]int{1, 2}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()

	// Build scoreMap
	scoreMap := make(map[int]map[int]*Score)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-23 * time.Hour), true},
		{2, 1, now.Add(-23 * time.Hour), true},
		{1, 2, now.Add(-22 * time.Hour), true},
		{2, 2, now.Add(-24 * time.Hour), true},
		{1, 3, now.Add(-20 * time.Hour), true},
	}

	// Initialize all teams with all child objectives
	for teamId := 1; teamId <= 3; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 2; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Now update with actual finished data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Add parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildRankingByTime(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildRankingByTime should not return an error")

	// Verify only teams that completed all children get points
	assert.Equal(t, 20, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 20 points (completed all children, rank 1)")
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 10 points (completed all children, rank 2)")
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 0 points (didn't complete all children)")

}

func TestHandleBingo(t *testing.T) {
	// This tests BINGO_N scoring where teams must complete N objectives to score
	// Currently not implemented (returns nil), so this is a placeholder test
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, func(id int) *repository.Objective {
			return &repository.Objective{
				Id: id,
			}
		}),
	}
	now := time.Now()

	// Build scoreMap
	scoreMap := make(map[int]map[int]*Score)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-23 * time.Hour), true},
		{3, 1, now.Add(-22 * time.Hour), true},
		{4, 2, now.Add(-24 * time.Hour), true},
		{5, 2, now.Add(-22 * time.Hour), true},
		{6, 3, now.Add(-22 * time.Hour), true},
	}

	for _, data := range childData {
		if scoreMap[data.teamId] == nil {
			scoreMap[data.teamId] = make(map[int]*Score)
		}
		scoreMap[data.teamId][data.objId] = &Score{
			ObjectiveId: data.objId,
			TeamId:      data.teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: data.objId,
					Finished:    data.finished,
					Timestamp:   data.timestamp,
				},
			},
		}
	}

	// Add parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleBingoN(2)(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoN should not return an error")

	// Note: handleBingoN is currently not implemented (returns nil), so we can't check results
	// Once implemented, check parent scores have correct points based on bingo completion
}

func TestHandleBingoBoardHorizontal(t *testing.T) {
	// This tests BINGO_BOARD scoring with a horizontal line completion (row 0)
	// Team completes objectives 1, 2, 3 which form the first row of a 3x3 grid
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	childData := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-24 * time.Hour)},
		{2, now.Add(-23 * time.Hour)},
		{3, now.Add(-22 * time.Hour)},
	}

	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	for _, data := range childData {
		scoreMap[1][data.objId] = &Score{
			ObjectiveId: data.objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: data.objId,
					Finished:    true,
					Timestamp:   data.timestamp,
				},
			},
		}
	}

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for completing horizontal bingo")
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should match the last completed objective in the bingo line")
}

func TestGetBingoVertical(t *testing.T) {
	// This tests BINGO_BOARD scoring with a vertical line completion (column 0)
	// Team completes objectives 1, 4, 7 which form the first column of a 3x3 grid
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	childData := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-24 * time.Hour)},
		{4, now.Add(-23 * time.Hour)},
		{7, now.Add(-22 * time.Hour)},
	}

	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	for _, data := range childData {
		scoreMap[1][data.objId] = &Score{
			ObjectiveId: data.objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: data.objId,
					Finished:    true,
					Timestamp:   data.timestamp,
				},
			},
		}
	}

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify vertical bingo completion is detected correctly
	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for completing vertical bingo")
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should match the last completed objective in the bingo line")
}

func TestHandleBingoBoardDiagonal(t *testing.T) {
	// This tests BINGO_BOARD scoring with a diagonal line completion
	// Team completes objectives 1, 5, 9 which form the main diagonal of a 3x3 grid
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	childData := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-24 * time.Hour)},
		{5, now.Add(-23 * time.Hour)},
		{9, now.Add(-22 * time.Hour)},
	}

	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	for _, data := range childData {
		scoreMap[1][data.objId] = &Score{
			ObjectiveId: data.objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: data.objId,
					Finished:    true,
					Timestamp:   data.timestamp,
				},
			},
		}
	}

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify diagonal bingo completion is detected correctly
	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for completing diagonal bingo")
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should match the last completed objective in the bingo line")
}

func TestHandleBingoBoardCorrectTime(t *testing.T) {
	// This tests that BINGO_BOARD correctly identifies the completion timestamp
	// The timestamp should be the LATEST child completion in the bingo line
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	children := []*repository.Objective{}
	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)

	var expectedTimestamp time.Time
	for i := range 3 {
		for j := range 3 {
			id := i*3 + j + 1
			children = append(children, &repository.Objective{Id: id, Extra: fmt.Sprintf("%d,%d", i, j)})
			timestamp := time.Now().Add(time.Duration(-id) * time.Hour)
			if id == 7 {
				expectedTimestamp = timestamp
			}
			scoreMap[1][id] = &Score{
				ObjectiveId: id,
				TeamId:      1,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: id,
						Finished:    true,
						Timestamp:   timestamp,
					},
				},
			}
		}
	}
	objective.Children = children

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify the bingo timestamp matches the last completed child in the line (objective 7)
	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points for bingo")
	assert.Equal(t, expectedTimestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Timestamp should be the latest timestamp in the bingo line (objective 7)")
}

func TestHandleBingoBoardCorrectRanking(t *testing.T) {
	// This tests that BINGO_BOARD correctly ranks teams based on completion time
	// Team with earlier completion of their bingo line should rank higher
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{30, 20, 10},
			},
		},
	}
	// Team 1: Completes row with timestamps[1,2,3] -> last at timestamps[3]
	// Team 2: Completes row with timestamps[0,1,2] -> last at timestamps[2] (earlier)
	// Team 2 should rank first
	timestamps := utils.Map([]int{1, 2, 3, 4, 5, 6, 7, 8, 9}, func(i int) time.Time {
		return time.Now().Add(time.Duration(-i) * time.Hour)
	})
	children := []*repository.Objective{}
	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)
	scoreMap[2] = make(map[int]*Score)

	for i := range 3 {
		objId := i + 1
		scoreMap[1][objId] = &Score{
			ObjectiveId: objId,
			TeamId:      1,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: objId,
					Finished:    true,
					Timestamp:   timestamps[i+1],
				},
			},
		}
		scoreMap[2][objId] = &Score{
			ObjectiveId: objId,
			TeamId:      2,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {
					ObjectiveId: objId,
					Finished:    true,
					Timestamp:   timestamps[i],
				},
			},
		}
		for j := range 3 {
			id := i*3 + j + 1
			children = append(children, &repository.Objective{Id: id, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}
	scoreMap[2][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      2,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify teams are ranked correctly by their bingo completion times
	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should receive first place points")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1 (earlier bingo completion)")
	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(1)), scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should receive second place points")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 2 (later bingo completion)")

	assert.Equal(t, timestamps[1].Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Team 1 timestamp should be timestamps[1] (latest in their bingo line)")
	assert.Equal(t, timestamps[0].Unix(), scoreMap[2][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Team 2 timestamp should be timestamps[0] (latest in their bingo line)")
}

func TestMultipleScoringPresetsOnUmbrellaObjective(t *testing.T) {
	// This tests that an umbrella objective can have multiple scoring methods applied:
	// 1. RANKED_TIME - ranks teams based on completion time of all children
	// 2. BONUS_PER_COMPLETION - gives bonus points for each completed child

	rankedPresetId := 100
	bonusPresetId := 200

	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:            rankedPresetId,
				ScoringMethod: repository.RANKED_COMPLETION,
				Points:        repository.ExtendingNumberSlice{50, 30, 10}, // Points for ranking 1st, 2nd, 3rd
			},
			{
				Id:            bonusPresetId,
				ScoringMethod: repository.BONUS_PER_COMPLETION,
				Points:        repository.ExtendingNumberSlice{15, 10, 5}, // Bonus for 1st, 2nd, 3rd+ child completed
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	now := time.Now()

	// Team 1: Completes all 4 children, finishes FIRST (earliest latest timestamp at -21h)
	// Team 2: Completes all 4 children, finishes SECOND (latest timestamp at -15h)
	// Team 3: Completes all 4 children, finishes THIRD (latest timestamp at -10h)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		// Team 1 - completes all 4, last one finishes at -21h
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-23 * time.Hour), true},
		{3, 1, now.Add(-22 * time.Hour), true},
		{4, 1, now.Add(-21 * time.Hour), true}, // Latest timestamp for team 1
		// Team 2 - completes all 4, last one finishes at -15h
		{1, 2, now.Add(-20 * time.Hour), true},
		{2, 2, now.Add(-18 * time.Hour), true},
		{3, 2, now.Add(-16 * time.Hour), true},
		{4, 2, now.Add(-15 * time.Hour), true}, // Latest timestamp for team 2
		// Team 3 - completes all 4, last one finishes at -10h
		{1, 3, now.Add(-14 * time.Hour), true},
		{2, 3, now.Add(-12 * time.Hour), true},
		{3, 3, now.Add(-11 * time.Hour), true},
		{4, 3, now.Add(-10 * time.Hour), true}, // Latest timestamp for team 3
	}

	// Initialize scoreMap for all teams and all child objectives
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 3; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)

		// Initialize all child objectives for this team (even unfinished ones)
		for childId := 1; childId <= 4; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					rankedPresetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
					bonusPresetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply actual completion data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[rankedPresetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[rankedPresetId].Timestamp = data.timestamp
		scoreMap[data.teamId][data.objId].PresetCompletions[bonusPresetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[bonusPresetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores with both presets
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				rankedPresetId: {ObjectiveId: objective.Id},
				bonusPresetId:  {ObjectiveId: objective.Id},
			},
		}
	}

	// Apply RANKED_COMPLETION scoring
	err := handleChildRankingByTime(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	// Apply BONUS_PER_COMPLETION scoring
	err = handleChildBonus(objective, objective.ScoringPresets[1], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	// Verify RANKED_COMPLETION results
	// Team 1 completes all children last at -21h -> rank 1 (all children done)
	// Team 2 completes 2 children last at -19h -> rank 2
	// Team 3 completes 1 child last at -18h -> rank 3
	assert.Equal(t, 50, scoreMap[1][objective.Id].PresetCompletions[rankedPresetId].Points, "Team 1 should rank 1st with 50 points")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[rankedPresetId].Rank, "Team 1 should have rank 1")

	assert.Equal(t, 30, scoreMap[2][objective.Id].PresetCompletions[rankedPresetId].Points, "Team 2 should rank 2nd with 30 points")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[rankedPresetId].Rank, "Team 2 should have rank 2")

	assert.Equal(t, 10, scoreMap[3][objective.Id].PresetCompletions[rankedPresetId].Points, "Team 3 should rank 3rd with 10 points")
	assert.Equal(t, 3, scoreMap[3][objective.Id].PresetCompletions[rankedPresetId].Rank, "Team 3 should have rank 3")

	// Verify BONUS_PER_COMPLETION results (applied to child objectives)
	// All teams complete 4 children, so bonus is distributed by completion order
	// Team 1: completes children at -24, -23, -22, -21 hours
	assert.Equal(t, 15, scoreMap[1][1].BonusPoints, "Team 1, child 1 should have 15 bonus (completed first)")
	assert.Equal(t, 10, scoreMap[1][2].BonusPoints, "Team 1, child 2 should have 10 bonus (completed second)")
	assert.Equal(t, 5, scoreMap[1][3].BonusPoints, "Team 1, child 3 should have 5 bonus (completed third)")
	assert.Equal(t, 5, scoreMap[1][4].BonusPoints, "Team 1, child 4 should have 5 bonus (completed fourth)")

	// Team 2: completes children at -20, -18, -16, -15 hours
	assert.Equal(t, 15, scoreMap[2][1].BonusPoints, "Team 2, child 1 should have 15 bonus (completed first)")
	assert.Equal(t, 10, scoreMap[2][2].BonusPoints, "Team 2, child 2 should have 10 bonus (completed second)")
	assert.Equal(t, 5, scoreMap[2][3].BonusPoints, "Team 2, child 3 should have 5 bonus (completed third)")
	assert.Equal(t, 5, scoreMap[2][4].BonusPoints, "Team 2, child 4 should have 5 bonus (completed fourth)")

	// Team 3: completes children at -14, -12, -11, -10 hours
	assert.Equal(t, 15, scoreMap[3][1].BonusPoints, "Team 3, child 1 should have 15 bonus (completed first)")
	assert.Equal(t, 10, scoreMap[3][2].BonusPoints, "Team 3, child 2 should have 10 bonus (completed second)")
	assert.Equal(t, 5, scoreMap[3][3].BonusPoints, "Team 3, child 3 should have 5 bonus (completed third)")
	assert.Equal(t, 5, scoreMap[3][4].BonusPoints, "Team 3, child 4 should have 5 bonus (completed fourth)")
	// Verify that both scoring methods are independent and don't interfere
	// The parent objective should have separate PresetCompletion entries for each preset
	assert.NotNil(t, scoreMap[1][objective.Id].PresetCompletions[rankedPresetId], "Ranked preset should exist")
	assert.NotNil(t, scoreMap[1][objective.Id].PresetCompletions[bonusPresetId], "Bonus preset should exist")

	// Verify the presets are truly separate (bonus preset doesn't set Points/Rank on parent)
	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[bonusPresetId].Points, "Bonus preset shouldn't set points on parent objective")
	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[bonusPresetId].Rank, "Bonus preset shouldn't set rank on parent objective")
}

func TestHandleChildRankingByNumber(t *testing.T) {
	// This tests MAX_CHILD_NUMBER_SUM scoring where teams are ranked by the sum of Number values
	// from child objectives (e.g., total atlas completion percentage, total boss kills, etc.)
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50, 25},
			},
		},
		Children: utils.Map([]int{1, 2, 3}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	// Test data:
	// Team 1: child numbers are 10, 20, 30 = 60 total (rank 1)
	// Team 2: child numbers are 15, 15, 15 = 45 total (rank 2)
	// Team 3: child numbers are 5, 10, 15 = 30 total (rank 3, tied with team 4)
	// Team 4: child numbers are 10, 10, 10 = 30 total (rank 3, tied with team 3)
	// Team 5: child numbers are 5, 5, 5 = 15 total (rank 4)
	childData := []struct {
		objId, teamId int
		number        int
	}{
		{1, 1, 10}, {2, 1, 20}, {3, 1, 30},
		{1, 2, 15}, {2, 2, 15}, {3, 2, 15},
		{1, 3, 5}, {2, 3, 10}, {3, 3, 15},
		{1, 4, 10}, {2, 4, 10}, {3, 4, 10},
		{1, 5, 5}, {2, 5, 5}, {3, 5, 5},
	}

	// Initialize scoreMap for all teams and all child objectives
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 5; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 3; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Number:      0,
					},
				},
			}
		}
	}

	// Apply the number data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Number = data.number
	}

	// Initialize parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	// Apply the ranking
	err := handleChildRankingByNumber(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	// Verify results - teams ranked by sum of child Number values
	// Team 1: 60 total -> rank 1, 100 points
	assert.Equal(t, 60, scoreMap[1][objective.Id].PresetCompletions[presetId].Number, "Team 1 should have 60 total")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1")
	assert.Equal(t, 100, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 100 points")

	// Team 2: 45 total -> rank 2, 75 points
	assert.Equal(t, 45, scoreMap[2][objective.Id].PresetCompletions[presetId].Number, "Team 2 should have 45 total")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 2")
	assert.Equal(t, 75, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 75 points")

	// Team 3 & 4: 30 total (tied) -> both rank 3, both get same points
	assert.Equal(t, 30, scoreMap[3][objective.Id].PresetCompletions[presetId].Number, "Team 3 should have 30 total")
	assert.Equal(t, 3, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 3 (tied)")
	assert.Equal(t, 50, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 50 points (rank 3)")

	assert.Equal(t, 30, scoreMap[4][objective.Id].PresetCompletions[presetId].Number, "Team 4 should have 30 total")
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 3 (tied)")
	assert.Equal(t, 50, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 50 points (rank 3, same as team 3)")

	// Team 5: 15 total -> rank 4 (after tied pair), points advance to position 4
	assert.Equal(t, 15, scoreMap[5][objective.Id].PresetCompletions[presetId].Number, "Team 5 should have 15 total")
	assert.Equal(t, 4, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank, "Team 5 should have rank 4")
	assert.Equal(t, 25, scoreMap[5][objective.Id].PresetCompletions[presetId].Points, "Team 5 should have 25 points (rank 4)")
}

func TestHandleChildRankingByTimeWithRequiredChildCompletions(t *testing.T) {
	// This tests RANKED_COMPLETION with Extra["required_child_completions"]
	// Teams only score if they complete the specified number of children (not necessarily all)
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50},
				Extra: map[string]string{
					"required_child_completions": "2", // Only need 2 out of 4 children
				},
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	now := time.Now()
	// Team 1: Completes 2 children, finishes at -20h (latest) -> rank 1
	// Team 2: Completes 2 children, finishes at -15h (latest) -> rank 2
	// Team 3: Completes 4 children, finishes at -10h (latest) -> rank 3
	// Team 4: Completes 1 child -> doesn't score (not enough completions)
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-20 * time.Hour), true}, // Team 1's latest
		{1, 2, now.Add(-18 * time.Hour), true},
		{2, 2, now.Add(-15 * time.Hour), true}, // Team 2's latest
		{1, 3, now.Add(-14 * time.Hour), true},
		{2, 3, now.Add(-12 * time.Hour), true},
		{3, 3, now.Add(-11 * time.Hour), true},
		{4, 3, now.Add(-10 * time.Hour), true}, // Team 3's latest
		{1, 4, now.Add(-22 * time.Hour), true}, // Team 4 only completes 1
	}

	// Initialize scoreMap
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 4; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 4; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply actual completion data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildRankingByTime(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildRankingByTime should not return an error")

	// Verify results - only teams with exactly 2 completions score
	// Teams are sorted: Team 3 (4 completions), Team 1 (2 completions, earlier), Team 2 (2 completions, later), Team 4 (1 completion)
	// Points are assigned based on position in sorted array, but only teams with exactly 2 completions get marked finished
	assert.Equal(t, 4, scoreMap[3][objective.Id].PresetCompletions[presetId].Number, "Team 3 should have 4 completions")
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 0 points (completed more than required)")
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank, "Team 3 should have rank 0")
	assert.False(t, scoreMap[3][objective.Id].PresetCompletions[presetId].Finished, "Team 3 shouldn't be finished (exceeded requirement)")

	assert.Equal(t, 2, scoreMap[1][objective.Id].PresetCompletions[presetId].Number, "Team 1 should have 2 completions")
	assert.Equal(t, 75, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 75 points (position 2 in sorted array)")
	assert.Equal(t, 2, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 2 (position in sorted array)")
	assert.True(t, scoreMap[1][objective.Id].PresetCompletions[presetId].Finished, "Team 1 should be marked finished")

	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Number, "Team 2 should have 2 completions")
	assert.Equal(t, 50, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 50 points (position 3 in sorted array)")
	assert.Equal(t, 3, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 3 (position in sorted array)")
	assert.True(t, scoreMap[2][objective.Id].PresetCompletions[presetId].Finished, "Team 2 should be marked finished")

	assert.Equal(t, 1, scoreMap[4][objective.Id].PresetCompletions[presetId].Number, "Team 4 should have 1 completion")
	assert.Equal(t, 0, scoreMap[4][objective.Id].PresetCompletions[presetId].Points, "Team 4 should have 0 points (not enough completions)")
	assert.Equal(t, 0, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank, "Team 4 should have rank 0")
	assert.False(t, scoreMap[4][objective.Id].PresetCompletions[presetId].Finished, "Team 4 shouldn't be finished")
}

func TestHandleChildRankingByTimeWithRequiredChildCompletionsPercent(t *testing.T) {
	// This tests RANKED_COMPLETION with Extra["required_child_completions_percent"]
	// Teams score if they complete at least the specified percentage of children
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50},
				Extra: map[string]string{
					"required_child_completions_percent": "50", // Need 50% of 4 children = 2
				},
			},
		},
		Children: utils.Map([]int{1, 2, 3, 4}, func(id int) *repository.Objective {
			return &repository.Objective{Id: id}
		}),
	}

	now := time.Now()
	// Team 1: Completes 3 children (75%) -> should score
	// Team 2: Completes 2 children (50% exactly) -> should score
	// Team 3: Completes 1 child (25%) -> should not score
	childData := []struct {
		objId, teamId int
		timestamp     time.Time
		finished      bool
	}{
		{1, 1, now.Add(-24 * time.Hour), true},
		{2, 1, now.Add(-22 * time.Hour), true},
		{3, 1, now.Add(-20 * time.Hour), true}, // Team 1's latest (3 completions)
		{1, 2, now.Add(-18 * time.Hour), true},
		{2, 2, now.Add(-15 * time.Hour), true}, // Team 2's latest (2 completions)
		{1, 3, now.Add(-10 * time.Hour), true}, // Team 3 only 1 completion
	}

	// Initialize scoreMap
	scoreMap := make(map[int]map[int]*Score)
	for teamId := 1; teamId <= 3; teamId++ {
		scoreMap[teamId] = make(map[int]*Score)
		for childId := 1; childId <= 4; childId++ {
			scoreMap[teamId][childId] = &Score{
				ObjectiveId: childId,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: childId,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply actual completion data
	for _, data := range childData {
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Finished = data.finished
		scoreMap[data.teamId][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores
	for teamId := range scoreMap {
		scoreMap[teamId][objective.Id] = &Score{
			ObjectiveId: objective.Id,
			TeamId:      teamId,
			PresetCompletions: map[int]*PresetCompletion{
				presetId: {ObjectiveId: objective.Id},
			},
		}
	}

	err := handleChildRankingByTime(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleChildRankingByTime should not return an error")

	// Verify results - only teams with exactly 2 completions (50% of 4) should score
	// Teams are sorted: Team 1 (3 completions), Team 2 (2 completions), Team 3 (1 completion)
	// Points are assigned based on position in sorted array
	assert.Equal(t, 3, scoreMap[1][objective.Id].PresetCompletions[presetId].Number, "Team 1 should have 3 completions")
	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 0 points (exceeded exact requirement)")
	assert.False(t, scoreMap[1][objective.Id].PresetCompletions[presetId].Finished, "Team 1 shouldn't be finished (more than required 50%)")

	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Number, "Team 2 should have 2 completions")
	assert.Equal(t, 75, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 75 points (position 2 in sorted array)")
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 2")
	assert.True(t, scoreMap[2][objective.Id].PresetCompletions[presetId].Finished, "Team 2 should be finished (exactly 50%)")

	assert.Equal(t, 1, scoreMap[3][objective.Id].PresetCompletions[presetId].Number, "Team 3 should have 1 completion")
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Points, "Team 3 should have 0 points (below 50%)")
	assert.False(t, scoreMap[3][objective.Id].PresetCompletions[presetId].Finished, "Team 3 shouldn't be finished")
}

func TestHandleBingoBoardWithRequiredNumberOfBingos(t *testing.T) {
	// This tests BINGO_BOARD with Extra["required_number_of_bingos"]
	// Teams must complete multiple bingo lines to score
	presetId := 100
	objective := &repository.Objective{
		Id: 10,
		ScoringPresets: []*repository.ScoringPreset{
			{
				Id:     presetId,
				Points: repository.ExtendingNumberSlice{100, 75, 50},
				Extra: map[string]string{
					"required_number_of_bingos": "2", // Need 2 bingos to score
				},
			},
		},
	}
	// Create 3x3 grid
	children := []*repository.Objective{}
	for i := range 3 {
		for j := range 3 {
			children = append(children, &repository.Objective{Id: i*3 + j + 1, Extra: fmt.Sprintf("%d,%d", i, j)})
		}
	}
	objective.Children = children

	now := time.Now()
	scoreMap := make(map[int]map[int]*Score)
	scoreMap[1] = make(map[int]*Score)
	scoreMap[2] = make(map[int]*Score)

	// Team 1: Completes first row (1,2,3) and first column (1,4,7) = 2 bingos
	// Row completion time: max(1,2,3) = objective 3 at -26h
	// Column completion time: max(1,4,7) = objective 7 at -20h
	// Bingo timestamp: min(-26h, -20h) = -26h (earliest bingo completion)
	team1Completions := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-30 * time.Hour)},
		{2, now.Add(-28 * time.Hour)},
		{3, now.Add(-26 * time.Hour)}, // Row completes here (earliest of two bingos)
		{4, now.Add(-24 * time.Hour)},
		{7, now.Add(-20 * time.Hour)}, // Column completes here
	}

	// Team 2: Completes first row (1,2,3) only = 1 bingo (shouldn't score)
	team2Completions := []struct {
		objId     int
		timestamp time.Time
	}{
		{1, now.Add(-25 * time.Hour)},
		{2, now.Add(-23 * time.Hour)},
		{3, now.Add(-21 * time.Hour)}, // Latest for team 2
	}

	// Initialize all objectives for both teams
	for teamId := 1; teamId <= 2; teamId++ {
		for i := 1; i <= 9; i++ {
			scoreMap[teamId][i] = &Score{
				ObjectiveId: i,
				TeamId:      teamId,
				PresetCompletions: map[int]*PresetCompletion{
					presetId: {
						ObjectiveId: i,
						Finished:    false,
						Timestamp:   time.Time{},
					},
				},
			}
		}
	}

	// Apply team 1 completions
	for _, data := range team1Completions {
		scoreMap[1][data.objId].PresetCompletions[presetId].Finished = true
		scoreMap[1][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Apply team 2 completions
	for _, data := range team2Completions {
		scoreMap[2][data.objId].PresetCompletions[presetId].Finished = true
		scoreMap[2][data.objId].PresetCompletions[presetId].Timestamp = data.timestamp
	}

	// Initialize parent objective scores
	scoreMap[1][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      1,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}
	scoreMap[2][objective.Id] = &Score{
		ObjectiveId: objective.Id,
		TeamId:      2,
		PresetCompletions: map[int]*PresetCompletion{
			presetId: {ObjectiveId: objective.Id},
		},
	}

	err := handleBingoBoard(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err, "handleBingoBoard should not return an error")

	// Verify team 1 scores (has 2 bingos)
	assert.True(t, scoreMap[1][objective.Id].PresetCompletions[presetId].Finished, "Team 1 should be finished (has 2 bingos)")
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank, "Team 1 should have rank 1")
	assert.Equal(t, 100, scoreMap[1][objective.Id].PresetCompletions[presetId].Points, "Team 1 should have 100 points")
	assert.Equal(t, team1Completions[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix(), "Team 1 timestamp should be earliest bingo completion (row completes at objective 3)")

	// Verify team 2 doesn't score (only has 1 bingo)
	assert.False(t, scoreMap[2][objective.Id].PresetCompletions[presetId].Finished, "Team 2 shouldn't be finished (only 1 bingo)")
	assert.Equal(t, 0, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank, "Team 2 should have rank 0")
	assert.Equal(t, 0, scoreMap[2][objective.Id].PresetCompletions[presetId].Points, "Team 2 should have 0 points")
}
