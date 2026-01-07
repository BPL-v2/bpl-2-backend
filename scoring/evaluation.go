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
	repository.RANKED_COMPLETION:    handleChildRankingByTime,
	repository.BONUS_PER_COMPLETION: handleChildBonus,
	repository.BINGO_3:              handleBingoN(3),
	repository.BINGO_BOARD:          handleBingoBoard,
	repository.MAX_CHILD_NUMBER_SUM: handleChildRankingByNumber,
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

type GridFinish struct {
	Grid Tuple
	Time time.Time
}

func handleBingoBoard(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	numberOfBingosRequired := 1
	if val, ok := scoringPreset.Extra["required_number_of_bingos"]; ok {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			numberOfBingosRequired = parsed
		}
	}
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

	teamChildFinishes := make(map[int][]GridFinish)

	for teamId, teamScores := range scoreMap {
		for childId := range objectiveMap {
			childScore := teamScores[childId]
			if childScore != nil && childScore.Finished() {
				teamChildFinishes[teamId] = append(teamChildFinishes[teamId], GridFinish{Grid: gridCellMap[childId], Time: childScore.Timestamp()})
			}
		}
	}

	bingoScores := []*PresetCompletion{}
	for teamId, finishedGridCells := range teamChildFinishes {
		gridTimestamps := make(map[int]map[int]time.Time)
		for _, completion := range finishedGridCells {
			cellPos := completion.Grid
			if _, exists := gridTimestamps[cellPos.X]; !exists {
				gridTimestamps[cellPos.X] = make(map[int]time.Time)
			}
			gridTimestamps[cellPos.X][cellPos.Y] = completion.Time
		}
		if scoreMap[teamId] == nil || scoreMap[teamId][objective.Id] == nil || scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		completion := scoreMap[teamId][objective.Id].PresetCompletions[scoringPreset.Id]
		finishTime := getBingoCompletionTime(numberOfBingosRequired, gridTimestamps, gridSize)
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

func getBingoCompletionTime(numberOfBingosRequired int, girdTimestamps map[int]map[int]time.Time, gridSize int) time.Time {
	finishTime := int64(math.MaxInt64)
	rowTimes := map[int][]int64{}
	colTimes := map[int][]int64{}
	diag1Times := []int64{}
	diag2Times := []int64{}
	for x, row := range girdTimestamps {
		for y := range row {
			gridSize = utils.Max(gridSize, x, y)
		}
	}

	for x, row := range girdTimestamps {
		for y, timestamp := range row {
			rowTimes[x] = append(rowTimes[x], timestamp.UnixNano())
			colTimes[y] = append(colTimes[y], timestamp.UnixNano())
			if x == y {
				diag1Times = append(diag1Times, timestamp.UnixNano())
			}
			if x+y == gridSize-1 {
				diag2Times = append(diag2Times, timestamp.UnixNano())
			}
		}
	}
	bingoCount := 0
	for i := 0; i < gridSize; i++ {
		if len(rowTimes[i]) == gridSize {
			finishTime = utils.Min(utils.Max(rowTimes[i]...), finishTime)
			bingoCount++
		}
		if len(colTimes[i]) == gridSize {
			finishTime = utils.Min(utils.Max(colTimes[i]...), finishTime)
			bingoCount++
		}
	}
	if len(diag1Times) == gridSize {
		finishTime = utils.Min(utils.Max(diag1Times...), finishTime)
		bingoCount++
	}
	if len(diag2Times) == gridSize {
		finishTime = utils.Min(utils.Max(diag2Times...), finishTime)
		bingoCount++
	}
	if finishTime == int64(math.MaxInt64) || bingoCount < numberOfBingosRequired {
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

func handleChildRankingByTime(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	requiredChildCompletions := len(objective.Children)
	if val, ok := scoringPreset.Extra["required_child_completions"]; ok {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			requiredChildCompletions = parsed
		}
	}
	if val, ok := scoringPreset.Extra["required_child_completions_percent"]; ok {
		parsed, err := strconv.Atoi(val)
		if err == nil {
			requiredChildCompletions = (len(objective.Children) * parsed) / 100
		}
	}
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
		if completion.ObjectivesCompleted == requiredChildCompletions {
			comp.Finished = true
			comp.Points = int(scoringPreset.Points.Get(i))
			comp.Rank = i + 1
			comp.Timestamp = time.Unix(0, completion.LatestTimestamp)
		}
	}
	return nil
}

func handleChildRankingByNumber(objective *repository.Objective, scoringPreset *repository.ScoringPreset, aggregations ObjectiveTeamMatches, scoreMap map[int]map[int]*Score) error {
	teamCompletions := make(map[int]*TeamCompletion)
	for teamId := range scoreMap {
		teamCompletions[teamId] = &TeamCompletion{TeamId: teamId}
	}
	childIds := map[int]bool{}
	for _, child := range objective.Children {
		childIds[child.Id] = true
		for teamId, objectiveScores := range scoreMap {
			childScore := objectiveScores[child.Id]
			if childScore != nil {
				teamCompletions[teamId].ObjectivesCompleted += childScore.PresetCompletions[scoringPreset.Id].Number
			}
		}
	}
	rankedTeams := utils.Values(teamCompletions)
	sort.Slice(rankedTeams, func(i, j int) bool {
		return rankedTeams[i].ObjectivesCompleted > rankedTeams[j].ObjectivesCompleted
	})
	rank := 1
	for i, completion := range rankedTeams {
		if scoreMap[completion.TeamId] == nil || scoreMap[completion.TeamId][objective.Id] == nil || scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringPreset.Id] == nil {
			continue
		}
		comp := scoreMap[completion.TeamId][objective.Id].PresetCompletions[scoringPreset.Id]
		comp.Number = completion.ObjectivesCompleted
		comp.Points = int(scoringPreset.Points.Get(rank - 1))
		comp.Rank = rank
		if i+1 < len(rankedTeams) && rankedTeams[i+1].ObjectivesCompleted < completion.ObjectivesCompleted {
			rank++
		}
	}
	return nil
}
