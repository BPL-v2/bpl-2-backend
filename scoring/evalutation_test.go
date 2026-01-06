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
	assert.NoError(t, err)

	assert.Equal(t, 0, scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points)
}

func TestHandlePointsFromValue(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, int(value*float64(match1.Number)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, int(value*float64(match2.Number)), scoreMap[2][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, objective.ScoringPresets[0].PointCap, scoreMap[3][objective.Id].PresetCompletions[presetId].Points)
}

func TestHandleRankedTime(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points)
}
func TestHandleRankedValue(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points)
}
func TestHandleRankedReverse(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 10, scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 1, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 2, scoreMap[3][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 5, scoreMap[3][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 3, scoreMap[4][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 5, scoreMap[4][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, 0, scoreMap[5][objective.Id].PresetCompletions[presetId].Points)
}

func TestHandleChildBonus(t *testing.T) {
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
	assert.NoError(t, err)

	// Check BonusPoints were added to child scores
	assert.Equal(t, 10, scoreMap[1][1].BonusPoints)
	assert.Equal(t, 9, scoreMap[1][2].BonusPoints)
	assert.Equal(t, 5, scoreMap[1][3].BonusPoints)
	assert.Equal(t, 5, scoreMap[1][4].BonusPoints)
	assert.Equal(t, 5, scoreMap[1][5].BonusPoints)
	assert.Equal(t, 10, scoreMap[2][1].BonusPoints)
}

func TestHandleChildRanking(t *testing.T) {
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

	err := handleChildRanking(objective, objective.ScoringPresets[0], make(ObjectiveTeamMatches), scoreMap)
	assert.NoError(t, err)

	assert.Equal(t, 20, scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 10, scoreMap[2][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 0, scoreMap[3][objective.Id].PresetCompletions[presetId].Points)

}

func TestHandleBingo(t *testing.T) {
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
	assert.NoError(t, err)

	// Note: handleBingoN is currently not implemented (returns nil), so we can't check results
	// Once implemented, check parent scores have correct points based on bingo completion
}

func TestHandleBingoBoardHorizontal(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix())
}

func TestGetBingoVertical(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix())
}

func TestHandleBingoBoardDiagonal(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, childData[2].timestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix())
}

func TestHandleBingoBoardCorrectTime(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, expectedTimestamp.Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix())
}

func TestHandleBingoBoardCorrectRanking(t *testing.T) {
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
	assert.NoError(t, err)

	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(0)), scoreMap[1][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 1, scoreMap[1][objective.Id].PresetCompletions[presetId].Rank)
	assert.Equal(t, int(objective.ScoringPresets[0].Points.Get(1)), scoreMap[2][objective.Id].PresetCompletions[presetId].Points)
	assert.Equal(t, 2, scoreMap[2][objective.Id].PresetCompletions[presetId].Rank)

	assert.Equal(t, timestamps[1].Unix(), scoreMap[1][objective.Id].PresetCompletions[presetId].Timestamp.Unix())
	assert.Equal(t, timestamps[0].Unix(), scoreMap[2][objective.Id].PresetCompletions[presetId].Timestamp.Unix())
}
