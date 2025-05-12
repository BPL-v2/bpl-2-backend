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

const (
	OBJECTIVE ScoreType = "OBJECTIVE"
	CATEGORY  ScoreType = "CATEGORY"
)

type Score struct {
	Type      ScoreType
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
	if s.Type == OBJECTIVE {
		return "O-" + strconv.Itoa(s.Id) + "-" + strconv.Itoa(s.TeamId)
	} else {
		return "C-" + strconv.Itoa(s.Id) + "-" + strconv.Itoa(s.TeamId)
	}
}

var scoreEvaluationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "score_evaluation_duration_s",
	Help: "Duration of Evaluation step during scoring",
	Buckets: []float64{
		0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10,
	},
})

func EvaluateAggregations(category *repository.ScoringCategory, aggregations ObjectiveTeamMatches) ([]*Score, error) {
	timer := prometheus.NewTimer(scoreEvaluationDuration)
	defer timer.ObserveDuration()
	scores := make([]*Score, 0)

	for _, objective := range category.Objectives {
		if objective.ScoringPreset != nil {
			fun, ok := objectiveScoringFunctions[objective.ScoringPreset.ScoringMethod]
			if ok {
				objScores, err := fun(objective, aggregations)
				if err != nil {
					return nil, err
				}
				scores = append(scores, objScores...)
			}
		}
	}

	if category.ScoringPreset != nil {
		fun, ok := categoryScoringFunctions[category.ScoringPreset.ScoringMethod]
		if ok {
			categoryScores, err := fun(category, scores)
			if err != nil {
				return nil, err
			}
			scores = append(scores, categoryScores...)
		}
	}

	for _, subCategory := range category.SubCategories {
		subScores, err := EvaluateAggregations(subCategory, aggregations)
		if err != nil {
			return nil, err
		}
		scores = append(scores, subScores...)
	}

	return scores, nil
}

type TeamCompletion struct {
	TeamId              int
	ObjectivesCompleted int
	LatestTimestamp     time.Time
}

var objectiveScoringFunctions = map[repository.ScoringMethod]func(objective *repository.Objective, aggregations ObjectiveTeamMatches) ([]*Score, error){
	repository.PRESENCE:          handlePresence,
	repository.RANKED_TIME:       handleRankedTime,
	repository.RANKED_VALUE:      handleRankedValue,
	repository.RANKED_REVERSE:    handleRankedReverse,
	repository.POINTS_FROM_VALUE: handlePointsFromValue,
}

var categoryScoringFunctions = map[repository.ScoringMethod]func(category *repository.ScoringCategory, childScores []*Score) ([]*Score, error){
	repository.RANKED_COMPLETION:    handleCategoryRanking,
	repository.BONUS_PER_COMPLETION: handleCategoryBonus,
}

func handlePointsFromValue(objective *repository.Objective, aggregations ObjectiveTeamMatches) ([]*Score, error) {
	scores := make([]*Score, 0)
	for teamId, match := range aggregations[objective.Id] {
		score := &Score{
			Type:      OBJECTIVE,
			Id:        objective.Id,
			TeamId:    teamId,
			UserId:    match.UserId,
			Timestamp: match.Timestamp,
			Number:    match.Number,
		}
		if match.Finished {
			score.Finished = true
			score.Points = int(objective.ScoringPreset.Points.Get(0) * float64(match.Number))
		}
		scores = append(scores, score)
	}
	return scores, nil
}

func handlePresence(objective *repository.Objective, aggregations ObjectiveTeamMatches) ([]*Score, error) {
	scores := make([]*Score, 0)
	for teamId, match := range aggregations[objective.Id] {
		score := &Score{
			Type:      OBJECTIVE,
			Id:        objective.Id,
			TeamId:    teamId,
			UserId:    match.UserId,
			Timestamp: match.Timestamp,
			Number:    match.Number,
		}
		if match.Finished {
			score.Finished = true
			score.Points = int(objective.ScoringPreset.Points.Get(0))
		}
		scores = append(scores, score)
	}

	return scores, nil
}

func handleRankedTime(objective *repository.Objective, aggregations ObjectiveTeamMatches) ([]*Score, error) {
	rankFun := func(a, b *Match) bool {
		if a.Finished && b.Finished {
			if a.Timestamp.Equal(b.Timestamp) {
				return a.Number > b.Number
			}
			return a.Timestamp.Before(b.Timestamp)
		}
		return a.Finished
	}
	return handleRanked(objective, aggregations, rankFun)
}

func handleRankedValue(objective *repository.Objective, aggregations ObjectiveTeamMatches) ([]*Score, error) {
	rankFun := func(a, b *Match) bool {
		return a.Number > b.Number
	}
	return handleRanked(objective, aggregations, rankFun)
}

func handleRankedReverse(objective *repository.Objective, aggregations ObjectiveTeamMatches) ([]*Score, error) {
	rankFun := func(a, b *Match) bool {
		return a.Number < b.Number
	}
	return handleRanked(objective, aggregations, rankFun)
}

func handleRanked(objective *repository.Objective, aggregations ObjectiveTeamMatches, rankFun func(*Match, *Match) bool) ([]*Score, error) {
	scores := make([]*Score, 0)
	matches := make([]*Match, 0)
	for _, match := range aggregations[objective.Id] {
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool { return rankFun(matches[i], matches[j]) })
	for i, match := range matches {
		score := &Score{
			Type:      OBJECTIVE,
			Id:        objective.Id,
			TeamId:    match.TeamId,
			UserId:    match.UserId,
			Timestamp: match.Timestamp,
			Number:    match.Number,
		}
		if match.Finished {
			score.Finished = true
			score.Rank = i + 1
			score.Points = int(objective.ScoringPreset.Points.Get(i))
		}
		scores = append(scores, score)

	}

	return scores, nil
}

func handleCategoryBonus(category *repository.ScoringCategory, objectiveScores []*Score) ([]*Score, error) {
	scores := make([]*Score, 0)
	finishCounts := make(map[int]int)
	teamIds := make(map[int]bool)
	for _, score := range objectiveScores {
		if score.Finished {
			finishCounts[score.TeamId]++
		}
		teamIds[score.TeamId] = true
	}
	for teamId, _ := range teamIds {
		points := 0
		for i := 0; i < finishCounts[teamId]; i++ {
			points += int(category.ScoringPreset.Points.Get(i))
		}
		score := &Score{
			Type:      CATEGORY,
			Id:        category.Id,
			TeamId:    teamId,
			Points:    points,
			Timestamp: time.Now(),
			Number:    finishCounts[teamId],
			Finished:  finishCounts[teamId] == len(category.Objectives),
		}
		scores = append(scores, score)
	}

	return scores, nil
}

func handleCategoryRanking(category *repository.ScoringCategory, objectiveScores []*Score) ([]*Score, error) {
	// count the number of objectives completed by each team
	teamCompletions := make(map[int]TeamCompletion)
	for _, score := range objectiveScores {
		if score.Finished {
			tc := teamCompletions[score.TeamId]
			if score.Timestamp.After(tc.LatestTimestamp) {
				tc.LatestTimestamp = score.Timestamp
			}
			tc.TeamId = score.TeamId
			tc.ObjectivesCompleted++
			teamCompletions[score.TeamId] = tc
		}
	}

	rankedTeams := utils.Values(teamCompletions)
	sort.Slice(rankedTeams, func(i, j int) bool {
		if rankedTeams[i].ObjectivesCompleted == rankedTeams[j].ObjectivesCompleted {
			return rankedTeams[i].LatestTimestamp.Before(rankedTeams[j].LatestTimestamp)
		}
		return rankedTeams[i].ObjectivesCompleted > rankedTeams[j].ObjectivesCompleted
	})
	categoryScores := make([]*Score, 0)
	for i, completion := range rankedTeams {
		score := &Score{
			Type:      CATEGORY,
			Id:        category.Id,
			TeamId:    completion.TeamId,
			Timestamp: completion.LatestTimestamp,
			Number:    completion.ObjectivesCompleted,
		}
		if completion.ObjectivesCompleted == len(category.Objectives) {
			score.Finished = true
			score.Points = int(category.ScoringPreset.Points.Get(i))
			score.Rank = i + 1
		}

		categoryScores = append(categoryScores, score)
	}
	return categoryScores, nil
}
