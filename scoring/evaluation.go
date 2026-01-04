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

type Score struct {
	Id           int
	Points       int
	TeamId       int
	UserId       int
	Rank         int
	Timestamp    time.Time
	Number       int
	Finished     bool
	HideProgress bool
}

func (s *Score) CanShowTo(teamId int) bool {
	return (s.TeamId == teamId) || s.Finished || !s.HideProgress
}

var scoreEvaluationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "score_evaluation_duration_s",
	Help: "Duration of Evaluation step during scoring",
	Buckets: []float64{
		0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10,
	},
})

func EvaluateAggregations(objective *repository.Objective, aggregations ObjectiveTeamMatches) ([]*Score, error) {
	timer := prometheus.NewTimer(scoreEvaluationDuration)
	defer timer.ObserveDuration()
	scores := make([]*Score, 0)
	for _, childObjective := range objective.Children {
		childScores, err := EvaluateAggregations(childObjective, aggregations)
		if err != nil {
			return nil, err
		}
		scores = append(scores, childScores...)
	}

	if objective.ScoringPreset != nil {
		if fun, ok := scoringFunctions[objective.ScoringPreset.ScoringMethod]; ok {
			categoryScores, err := fun(objective, aggregations, scores)
			if err != nil {
				return nil, err
			}
			for _, score := range categoryScores {
				score.HideProgress = objective.HideProgress
			}
			scores = append(scores, categoryScores...)
		}
	}

	return scores, nil
}

type TeamCompletion struct {
	TeamId              int
	ObjectivesCompleted int
	LatestTimestamp     time.Time
}

var scoringFunctions = map[repository.ScoringMethod]func(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error){
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

func handlePointsFromValue(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	scores := make([]*Score, 0)
	for teamId, match := range aggregations[objective.Id] {
		score := &Score{
			Id:        objective.Id,
			TeamId:    teamId,
			UserId:    match.UserId,
			Timestamp: match.Timestamp,
			Number:    match.Number,
			Points:    int(objective.ScoringPreset.Points.Get(0) * float64(match.Number)),
			Finished:  match.Finished,
		}
		if objective.ScoringPreset.PointCap != 0 && score.Points > objective.ScoringPreset.PointCap {
			score.Points = objective.ScoringPreset.PointCap
		}
		scores = append(scores, score)
	}
	return scores, nil
}

func handlePresence(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	scores := make([]*Score, 0)
	for teamId, match := range aggregations[objective.Id] {
		score := &Score{
			Id:        objective.Id,
			TeamId:    teamId,
			UserId:    match.UserId,
			Timestamp: match.Timestamp,
			Number:    match.Number,
			Finished:  match.Finished,
		}
		if match.Finished {
			score.Points = int(objective.ScoringPreset.Points.Get(0))
		}
		scores = append(scores, score)
	}

	return scores, nil
}

func handleBingoN(n int) func(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	// Handles a category of collection goals where a team must finish n goals to score, but does not get more points for finishing more than n.
	return func(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
		sc := make(map[int][]*Score, 0)

		for _, score := range childScores {
			if score.Points > 0 {
				sc[score.TeamId] = append(sc[score.TeamId], score)
			}
		}
		timeToFinish := make(map[int]time.Time, 0)
		for teamId, scores := range sc {
			if len(scores) < n {
				continue
			}
			sort.Slice(scores, func(i, j int) bool {
				return scores[i].Timestamp.Before(scores[j].Timestamp)
			})
			timeToFinish[teamId] = scores[n-1].Timestamp
			for i := n; i < len(scores); i++ {
				scores[i].Points = 0
			}
		}
		finishes := make([]TeamCompletion, 0, len(timeToFinish))
		for teamId, ts := range timeToFinish {
			finishes = append(finishes, TeamCompletion{TeamId: teamId, LatestTimestamp: ts})
		}
		sort.Slice(finishes, func(i, j int) bool {
			return finishes[i].LatestTimestamp.Before(finishes[j].LatestTimestamp)
		})

		placements := make(map[int]int, len(finishes))
		scores := make([]*Score, 0)
		rank := 1
		for i, f := range finishes {
			if i > 0 && f.LatestTimestamp.After(finishes[i-1].LatestTimestamp) {
				rank = i + 1
			}
			placements[f.TeamId] = rank
			scores = append(scores, &Score{
				Id:        objective.Id,
				TeamId:    f.TeamId,
				Timestamp: f.LatestTimestamp,
				Number:    n,
				Finished:  true,
				Points:    int(objective.ScoringPreset.Points.Get(rank - 1)),
				Rank:      rank,
			})
		}
		return scores, nil
	}

}

func handleRankedTime(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	rankFun := func(a, b *Match) bool {
		if a.Finished && b.Finished {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Finished
	}
	return handleRanked(objective, aggregations, rankFun)
}

func handleRankedValue(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	rankFun := func(a, b *Match) bool {
		if a.Number == b.Number {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Number > b.Number
	}
	return handleRanked(objective, aggregations, rankFun)
}

func handleRankedReverse(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	rankFun := func(a, b *Match) bool {
		if a.Number == b.Number {
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Number < b.Number
	}
	return handleRanked(objective, aggregations, rankFun)
}

func isTiedWithNext(index int, matches []*Match, rankFun func(a, b *Match) bool) bool {
	if index >= len(matches)-1 {
		return false
	}
	return rankFun(matches[index], matches[index+1]) == rankFun(matches[index+1], matches[index])
}

func handleRanked(objective *repository.Objective, aggregations ObjectiveTeamMatches, rankFun func(a, b *Match) bool) ([]*Score, error) {
	scores := make([]*Score, 0)
	matches := make([]*Match, 0)
	for _, match := range aggregations[objective.Id] {
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool { return rankFun(matches[i], matches[j]) })
	i := 0
	for j, match := range matches {
		score := &Score{
			Id:        objective.Id,
			TeamId:    match.TeamId,
			UserId:    match.UserId,
			Timestamp: match.Timestamp,
			Number:    match.Number,
			Finished:  match.Finished,
		}
		if match.Finished {
			score.Rank = i + 1
			score.Points = int(objective.ScoringPreset.Points.Get(i))
		}
		scores = append(scores, score)
		if !isTiedWithNext(j, matches, rankFun) {
			i++
		}

	}

	return scores, nil
}

type Tuple struct {
	X int
	Y int
}

func handleBingoBoard(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
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

	teamChildFinishes := make(map[int][]*Score)
	for _, score := range childScores {
		if score.Finished {
			teamChildFinishes[score.TeamId] = append(teamChildFinishes[score.TeamId], score)
		}
	}

	bingoScores := []*Score{}
	for teamId, finishedGridCells := range teamChildFinishes {
		gridToScores := make(map[int]map[int]*Score)
		for _, score := range finishedGridCells {
			cellPos, ok := gridCellMap[score.Id]
			if !ok {
				continue
			}
			if _, exists := gridToScores[cellPos.X]; !exists {
				gridToScores[cellPos.X] = make(map[int]*Score)
			}
			gridToScores[cellPos.X][cellPos.Y] = score
		}
		score := &Score{
			Id:     objective.Id,
			TeamId: teamId,
			Points: 0,
			Number: 0,
		}
		finishTime := getBingoCompletionTime(gridToScores, gridSize)
		if !finishTime.IsZero() {
			score.Finished = true
			score.Timestamp = finishTime
		}
		bingoScores = append(bingoScores, score)
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
			score.Points = int(objective.ScoringPreset.Points.Get(rank - 1))
		}
	}
	return append(bingoScores, childScores...), nil
}

func getBingoCompletionTime(scores map[int]map[int]*Score, gridSize int) time.Time {
	finishTime := int64(math.MaxInt64)
	rowTimes := map[int][]int64{}
	colTimes := map[int][]int64{}
	diag1Times := []int64{}
	diag2Times := []int64{}
	for x, row := range scores {
		for y := range row {
			gridSize = utils.Max(gridSize, x, y)
		}
	}
	for x, row := range scores {
		for y, score := range row {
			if score.Finished {
				rowTimes[x] = append(rowTimes[x], score.Timestamp.UnixNano())
				colTimes[y] = append(colTimes[y], score.Timestamp.UnixNano())
				if x == y {
					diag1Times = append(diag1Times, score.Timestamp.UnixNano())
				}
				if x+y == gridSize-1 {
					diag2Times = append(diag2Times, score.Timestamp.UnixNano())
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

func handleChildBonus(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	scores := make([]*Score, 0)
	finishCounts := make(map[int]int)
	teamIds := make(map[int]bool)
	for _, score := range childScores {
		if score.Finished {
			finishCounts[score.TeamId]++
		}
		teamIds[score.TeamId] = true
	}
	childIds := utils.Map(objective.Children, func(o *repository.Objective) int { return o.Id })
	for teamId := range teamIds {
		teamChildScores := utils.Filter(childScores, func(s *Score) bool {
			return s.TeamId == teamId && utils.Contains(childIds, s.Id) && s.Finished
		})
		sort.Slice(teamChildScores, func(i, j int) bool {
			return teamChildScores[i].Timestamp.Before(teamChildScores[j].Timestamp)
		})
		latestTimestamp := time.Time{}
		for i, score := range teamChildScores {
			score.Points += int(objective.ScoringPreset.Points.Get(i))
			if score.Timestamp.After(latestTimestamp) {
				latestTimestamp = score.Timestamp
			}
		}
		score := &Score{
			Id:        objective.Id,
			TeamId:    teamId,
			Points:    0,
			Timestamp: latestTimestamp,
			Number:    finishCounts[teamId],
			Finished:  finishCounts[teamId] == len(objective.Children),
		}
		scores = append(scores, score)
	}
	return scores, nil
}

func handleChildRanking(objective *repository.Objective, aggregations ObjectiveTeamMatches, childScores []*Score) ([]*Score, error) {
	teamCompletions := make(map[int]TeamCompletion)
	childIds := map[int]bool{}
	for _, child := range objective.Children {
		childIds[child.Id] = true
	}
	for _, score := range childScores {
		if score.Finished && childIds[score.Id] {
			completion := teamCompletions[score.TeamId]
			if score.Timestamp.After(completion.LatestTimestamp) {
				completion.LatestTimestamp = score.Timestamp
			}
			completion.TeamId = score.TeamId
			completion.ObjectivesCompleted++
			teamCompletions[score.TeamId] = completion
		}
	}
	rankedTeams := utils.Values(teamCompletions)
	sort.Slice(rankedTeams, func(i, j int) bool {
		if rankedTeams[i].ObjectivesCompleted == rankedTeams[j].ObjectivesCompleted {
			return rankedTeams[i].LatestTimestamp.Before(rankedTeams[j].LatestTimestamp)
		}
		return rankedTeams[i].ObjectivesCompleted > rankedTeams[j].ObjectivesCompleted
	})
	scores := make([]*Score, 0)
	for i, completion := range rankedTeams {
		score := &Score{
			Id:        objective.Id,
			TeamId:    completion.TeamId,
			Timestamp: completion.LatestTimestamp,
			Number:    completion.ObjectivesCompleted,
		}
		if completion.ObjectivesCompleted == len(childIds) {
			score.Finished = true
			score.Points = int(objective.ScoringPreset.Points.Get(i))
			score.Rank = i + 1
		}

		scores = append(scores, score)
	}
	return scores, nil
}
