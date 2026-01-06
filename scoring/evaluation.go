package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"math"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ScoreType string

type PresetCompletion struct {
	Finished    bool
	Timestamp   time.Time
	Rank        int
	UserId      int
	Points      int
	Number      int
	ObjectiveId int
}

type Score struct {
	ObjectiveId       int
	TeamId            int
	PresetCompletions map[int]*PresetCompletion
	HideProgress      bool
	BonusPoints       int
}

func (s *Score) Finished() bool {
	for _, pc := range s.PresetCompletions {
		if !pc.Finished {
			return false
		}
	}
	return true
}

func (s *Score) Timestamp() time.Time {
	latest := time.Time{}
	if !s.Finished() {
		return latest
	}
	for _, pc := range s.PresetCompletions {
		if pc.Timestamp.After(latest) {
			latest = pc.Timestamp
		}
	}
	return latest
}

func (s *Score) Points() int {
	total := s.BonusPoints
	for _, pc := range s.PresetCompletions {
		total += pc.Points
	}
	return total
}

func (s *Score) CanShowTo(teamId int) bool {
	return (s.TeamId == teamId) || s.Finished() || !s.HideProgress
}

var scoreEvaluationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "score_evaluation_duration_s",
	Help: "Duration of Evaluation step during scoring",
	Buckets: []float64{
		0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10,
	},
})

func EvaluateAggregations(objective *repository.Objective, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	timer := prometheus.NewTimer(scoreEvaluationDuration)
	defer timer.ObserveDuration()
	for _, childObjective := range objective.Children {
		err := EvaluateAggregations(childObjective, aggregations, scoreMap)
		if err != nil {
			return err
		}
	}
	for _, preset := range objective.ScoringPresets {
		if fun, ok := scoringFunctions[preset.ScoringMethod]; ok {
			err := fun(objective, preset, aggregations, scoreMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type TeamCompletion struct {
	TeamId              int
	ObjectivesCompleted int
	LatestTimestamp     int64
}

var scoringFunctions = map[repository.ScoringMethod]func(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error{
	repository.PRESENCE:             handlePresence,
	repository.RANKED_TIME:          handleRankedTime,
	repository.RANKED_VALUE:         handleRankedValue,
	repository.RANKED_REVERSE:       handleRankedReverse,
	repository.POINTS_FROM_VALUE:    handlePointsFromValue,
	repository.RANKED_COMPLETION:    handleChildRanking,
	repository.BONUS_PER_COMPLETION: handleChildBonus,
	repository.BINGO_3:              handleBingoN(3),
	repository.BINGO_BOARD:          handleBingoBoard,
}

func handlePointsFromValue(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	for teamId, match := range aggregations[objective.Id] {
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id]
		completion.Number = match.Number
		completion.UserId = match.UserId
		completion.Finished = match.Finished
		completion.Timestamp = match.Timestamp
		completion.Points = int(scoringPreset.Points.Get(0) * float64(match.Number))
		if scoringPreset.PointCap != 0 && completion.Points > scoringPreset.PointCap {
			completion.Points = scoringPreset.PointCap
		}
	}
	return nil
}

func handlePresence(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	for teamId, match := range aggregations[objective.Id] {
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id]
		completion.Number = match.Number
		completion.UserId = match.UserId
		completion.Finished = match.Finished
		completion.Timestamp = match.Timestamp
		if match.Finished {
			completion.Points = int(scoringPreset.Points.Get(0))
		}
	}
	return nil
}
func handleBingoN(n int) func(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	// can't be assed to fix this right now
	return func(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
		return nil
	}
}

// func handleBingoN(n int) func(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) (map[int]map[int]*Score, error) {
// 	// Handles a category of collection goals where a team must finish n goals to score, but does not get more points for finishing more than n.
// 	return func(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) (map[int]map[int]*Score, error) {
// 		sc := make(map[int][]*Score, 0)

// 		for _, score := range childScores {
// 			if score.Points > 0 {
// 				sc[score.TeamId] = append(sc[score.TeamId], score)
// 			}
// 		}
// 		timeToFinish := make(map[int]time.Time, 0)
// 		for teamId, scores := range sc {
// 			if len(scores) < n {
// 				continue
// 			}
// 			sort.Slice(scores, func(i, j int) bool {
// 				return scores[i].Timestamp.Before(scores[j].Timestamp)
// 			})
// 			timeToFinish[teamId] = scores[n-1].Timestamp
// 			for i := n; i < len(scores); i++ {
// 				scores[i].Points = 0
// 			}
// 		}
// 		finishes := make([]TeamCompletion, 0, len(timeToFinish))
// 		for teamId, ts := range timeToFinish {
// 			finishes = append(finishes, TeamCompletion{TeamId: teamId, LatestTimestamp: ts})
// 		}
// 		sort.Slice(finishes, func(i, j int) bool {
// 			return finishes[i].LatestTimestamp.Before(finishes[j].LatestTimestamp)
// 		})

// 		placements := make(map[int]int, len(finishes))
// 		scores := make([]*Score, 0)
// 		rank := 1
// 		for i, f := range finishes {
// 			if i > 0 && f.LatestTimestamp.After(finishes[i-1].LatestTimestamp) {
// 				rank = i + 1
// 			}
// 			placements[f.TeamId] = rank
// 			scores = append(scores, &Score{
// 				ObjectiveId: objective.Id,
// 				TeamId:      f.TeamId,
// 				Timestamp:   f.LatestTimestamp,
// 				Number:      n,
// 				Finished:    true,
// 				Points:      int(scoringPreset.Points.Get(rank - 1)),
// 				Rank:        rank,
// 			})
// 		}
// 		return scores, nil
// 	}

// }

func handleRankedTime(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	rankFun := func(a, b *Match) bool {
		if a.Finished && b.Finished {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Finished
	}
	return handleRanked(objective, scoringPreset, aggregations, rankFun, scoreMap)
}

func handleRankedValue(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	rankFun := func(a, b *Match) bool {
		if a.Number == b.Number {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Number > b.Number
	}
	return handleRanked(objective, scoringPreset, aggregations, rankFun, scoreMap)
}

func handleRankedReverse(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	rankFun := func(a, b *Match) bool {
		if a.Number == b.Number {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Number < b.Number
	}
	return handleRanked(objective, scoringPreset, aggregations, rankFun, scoreMap)
}

func isTiedWithNext(index int, matches []*Match, rankFun func(a, b *Match) bool) bool {
	if index >= len(matches)-1 {
		return false
	}
	return rankFun(matches[index], matches[index+1]) == rankFun(matches[index+1], matches[index])
}

func handleRanked(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, rankFun func(a, b *Match) bool, scoreMap map[int]map[int]*Score) error {
	matches := make([]*Match, 0)
	for _, match := range aggregations[objective.Id] {
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool { return rankFun(matches[i], matches[j]) })
	i := 0
	for j, match := range matches {
		if scoreMap[match.TeamId] == nil || scoreMap[match.TeamId][objective.Id] == nil || scoreMap[match.TeamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		completion := scoreMap[match.TeamId][objective.Id].PresetCompletions[scoringPreset.Id]
		completion.UserId = match.UserId
		completion.Timestamp = match.Timestamp
		completion.Number = match.Number
		completion.Finished = match.Finished

		if match.Finished {
			completion.Rank = i + 1
			completion.Points = int(scoringPreset.Points.Get(i))
		}
		if !isTiedWithNext(j, matches, rankFun) {
			i++
		}
	}
	return nil
}

type Tuple struct {
	X int
	Y int
}

func handleBingoBoard(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	objectiveMap := make(map[int]*repository.Objective)
	for _, child := range objective.Children {
		objectiveMap[child.Id] = child
	}
	gridCellMap := make(map[int]Tuple)
	regex := regexp.MustCompile(`(\d+),(\d+)`)
	gridSize := 0
	for _, child := range objective.Children {
		matches := regex.FindStringSubmatch(child.Extra)
		if len(matches) == 3 {
			x, _ := strconv.Atoi(matches[1])
			y, _ := strconv.Atoi(matches[2])
			gridCellMap[child.Id] = Tuple{X: x, Y: y}
			gridSize = utils.Max(gridSize, x+1, y+1)
		}
	}

	teamChildFinishes := make(map[int][]*PresetCompletion)
	for teamId, teamScores := range scoreMap {
		for objectiveId, score := range teamScores {
			if _, ok := gridCellMap[objectiveId]; ok {
				completion := score.PresetCompletions[scoringPreset.Id]
				if completion != nil && completion.Finished {
					teamChildFinishes[teamId] = append(teamChildFinishes[teamId], completion)
				}
			}
		}
	}

	bingoScores := []*PresetCompletion{}
	for teamId, finishedGridCells := range teamChildFinishes {
		gridToScores := make(map[int]map[int]*PresetCompletion)
		for _, completion := range finishedGridCells {
			cellPos, ok := gridCellMap[completion.ObjectiveId]
			if !ok {
				continue
			}
			if _, exists := gridToScores[cellPos.X]; !exists {
				gridToScores[cellPos.X] = make(map[int]*PresetCompletion)
			}
			gridToScores[cellPos.X][cellPos.Y] = completion
		}
		// score := scoreMap[teamId][objective.Id]
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id]
		finishTime := getBingoCompletionTime(gridToScores, gridSize)
		if !finishTime.IsZero() {
			completion.Finished = true
			completion.Timestamp = finishTime
		}
		bingoScores = append(bingoScores, completion)
	}

	sort.Slice(bingoScores, func(i, j int) bool {
		if bingoScores[i].Finished != bingoScores[j].Finished {
			return bingoScores[i].Finished
		}
		return bingoScores[i].Timestamp.Before(bingoScores[j].Timestamp)
	})
	rank := 1
	for i, score := range bingoScores {
		if score.Finished {
			if i > 0 && (bingoScores[i-1].Finished && bingoScores[i-1].Timestamp.Before(score.Timestamp)) {
				rank = i + 1
			}
			score.Rank = rank
			score.Points = int(scoringPreset.Points.Get(rank - 1))
		}
	}
	return nil
}

func getBingoCompletionTime(completions map[int]map[int]*PresetCompletion, gridSize int) time.Time {
	finishTime := int64(math.MaxInt64)
	rowTimes := map[int][]int64{}
	colTimes := map[int][]int64{}
	diag1Times := []int64{}
	diag2Times := []int64{}
	for x, row := range completions {
		for y := range row {
			gridSize = utils.Max(gridSize, x, y)
		}
	}
	for x, row := range completions {
		for y, completion := range row {
			if completion.Finished {
				rowTimes[x] = append(rowTimes[x], completion.Timestamp.UnixNano())
				colTimes[y] = append(colTimes[y], completion.Timestamp.UnixNano())
				if x == y {
					diag1Times = append(diag1Times, completion.Timestamp.UnixNano())
				}
				if x+y == gridSize-1 {
					diag2Times = append(diag2Times, completion.Timestamp.UnixNano())
				}
			}
		}
	}
	for i := 0; i < gridSize; i++ {
		if len(rowTimes[i]) == gridSize {
			finishTime = utils.Min(utils.Max(rowTimes[i]...), finishTime)
		}
		if len(colTimes[i]) == gridSize {
			finishTime = utils.Min(utils.Max(colTimes[i]...), finishTime)
		}
	}
	if len(diag1Times) == gridSize {
		finishTime = utils.Min(utils.Max(diag1Times...), finishTime)
	}
	if len(diag2Times) == gridSize {
		finishTime = utils.Min(utils.Max(diag2Times...), finishTime)
	}
	if finishTime == int64(math.MaxInt64) {
		return time.Time{}
	}
	return time.Unix(0, finishTime)
}

func handleChildBonus(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	finishCounts := make(map[int]int)
	teamIds := make(map[int]bool)
	childIds := utils.Map(objective.Children, func(o *repository.Objective) int { return o.Id })
	teamChildScores := map[int][]*Score{}
	for teamId, objectiveScores := range scoreMap {
		teamIds[teamId] = true
		for _, id := range childIds {
			score := objectiveScores[id]
			if score != nil && score.Finished() {
				finishCounts[teamId]++
				teamChildScores[teamId] = append(teamChildScores[teamId], score)
			}
		}
		sort.Slice(teamChildScores[teamId], func(i, j int) bool {
			return teamChildScores[teamId][i].Timestamp().Before(teamChildScores[teamId][j].Timestamp())
		})
	}

	for teamId, childScores := range teamChildScores {
		latestTimestamp := time.Time{}
		for i, childScore := range childScores {
			childScore.BonusPoints += int(scoringPreset.Points.Get(i))
			currentTimestamp := childScore.Timestamp()
			if currentTimestamp.After(latestTimestamp) {
				latestTimestamp = currentTimestamp
			}
		}
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id]
		completion.Finished = finishCounts[teamId] == len(objective.Children)
		completion.Number = finishCounts[teamId]
		completion.Timestamp = latestTimestamp
	}
	return nil
}

func handleChildRanking(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	teamCompletions := make(map[int]*TeamCompletion)
	for teamId := range scoreMap {
		teamCompletions[teamId] = &TeamCompletion{TeamId: teamId}
	}
	childIds := map[int]bool{}
	for _, child := range objective.Children {
		childIds[child.Id] = true
		for teamId, objectiveScores := range scoreMap {
			childScore := objectiveScores[child.Id]
			if childScore != nil && childScore.Finished() {
				teamCompletions[teamId].ObjectivesCompleted++
				teamCompletions[teamId].LatestTimestamp = utils.Max(teamCompletions[teamId].LatestTimestamp, childScore.Timestamp().UnixNano())
			}
		}
	}
	rankedTeams := utils.Values(teamCompletions)
	sort.Slice(rankedTeams, func(i, j int) bool {
		if rankedTeams[i].ObjectivesCompleted == rankedTeams[j].ObjectivesCompleted {
			return rankedTeams[i].LatestTimestamp < rankedTeams[j].LatestTimestamp
		}
		return rankedTeams[i].ObjectivesCompleted > rankedTeams[j].ObjectivesCompleted
	})
	for i, completion := range rankedTeams {
		if scoreMap[completion.TeamId] == nil || scoreMap[completion.TeamId][objective.Id] == nil || scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		comp := scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringPreset.Id]
		comp.Number = completion.ObjectivesCompleted
		if completion.ObjectivesCompleted == len(childIds) {
			comp.Finished = true
			comp.Points = int(scoringPreset.Points.Get(i))
			comp.Rank = i + 1
		}
	}
	return nil
}
