package scoring

import (
	"bpl/repository"
	"bpl/utils"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ScoreType string

type Score struct {
	Id        int
	Points    int
	TeamId    int
	UserId    int
	Rank      int
	Timestamp time.Time
	Number    int
	Finished  bool
}

func (s *Score) Identifier() string {
	return strconv.Itoa(s.Id) + "-" + strconv.Itoa(s.TeamId)
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
	for teamId, _ := range teamIds {
		points := 0
		for i := range finishCounts[teamId] {
			points += int(objective.ScoringPreset.Points.Get(i))
		}
		score := &Score{
			Id:        objective.Id,
			TeamId:    teamId,
			Points:    points,
			Timestamp: time.Now(),
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
