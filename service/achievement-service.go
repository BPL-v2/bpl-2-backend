package service

import (
	"bpl/repository"
)

type AchievementService struct {
	achievementRepository *repository.AchievementRepository
	characterRepository   *repository.CharacterRepository
}

func NewAchievementService() *AchievementService {
	return &AchievementService{
		achievementRepository: repository.NewAchievementRepository(),
		characterRepository:   repository.NewCharacterRepository(),
	}
}

func (s *AchievementService) FindAllAchievements() ([]*repository.Achievement, error) {
	return s.achievementRepository.GetAllAchievements()
}

func (s *AchievementService) SaveAchievement(achievement *repository.Achievement) error {
	return s.achievementRepository.SaveAchievement(achievement)
}

func (s *AchievementService) FindAchievementsForUser(userId int) ([]*repository.Achievement, error) {
	return s.achievementRepository.GetAchievementsForUser(userId)
}

func (s *AchievementService) UpdateAchievements() error {
	character, err := s.characterRepository.GetAllHighestLevelCharactersForEachEventAndUser()
	if err != nil {
		return err
	}
	characterMap := make(map[int][]*repository.Character)
	for _, char := range character {
		if char.UserId != nil {
			characterMap[*char.UserId] = append(characterMap[*char.UserId], char)
		}
	}
	achievements := []*repository.Achievement{}
	for userId, chars := range characterMap {
		userAchievements := checkAchievements(chars)
		for _, achievementName := range userAchievements {
			achievement := &repository.Achievement{
				UserId: userId,
				Name:   achievementName,
			}
			achievements = append(achievements, achievement)
		}
	}
	return s.achievementRepository.SaveAchievements(achievements)
}

var baseClasses = map[string]bool{
	"Scion":    true,
	"Marauder": true,
	"Ranger":   true,
	"Witch":    true,
	"Shadow":   true,
	"Duelist":  true,
	"Templar":  true,
}

func checkAchievements(chars []*repository.Character) []repository.AchievementName {
	achievements := []repository.AchievementName{}
	for achievement := range repository.Achievements {
		switch achievement {
		case repository.AchievementReachedLvl90:
			if hasLevelNCharacter(90, chars) {
				achievements = append(achievements, achievement)
			}
		case repository.AchievementReachedLvl95:
			if hasLevelNCharacter(95, chars) {
				achievements = append(achievements, achievement)
			}
		case repository.AchievementReachedLvl100:
			if hasLevelNCharacter(100, chars) {
				achievements = append(achievements, achievement)
			}
		case repository.AchievementParticipated:
			if playedNLeagues(1, chars) {
				achievements = append(achievements, achievement)
			}
		case repository.AchievementPlayed5Leagues:
			if playedNLeagues(5, chars) {
				achievements = append(achievements, achievement)
			}
		case repository.AchievementPlayed10Leagues:
			if playedNLeagues(10, chars) {
				achievements = append(achievements, achievement)
			}
		case repository.AchievementPlayed5DifferentAscendancies:
			if playedNDifferentAscendancies(5, chars) {
				achievements = append(achievements, achievement)
			}
		case repository.AchievementPlayed10DifferentAscendancies:
			if playedNDifferentAscendancies(10, chars) {
				achievements = append(achievements, achievement)
			}
		default:
			continue
		}
	}
	return achievements
}

func hasLevelNCharacter(level int, chars []*repository.Character) bool {
	for _, char := range chars {
		if char.Level >= level {
			return true
		}
	}
	return false
}

func playedNLeagues(n int, chars []*repository.Character) bool {
	return len(chars) >= n
}

func playedNDifferentAscendancies(n int, chars []*repository.Character) bool {
	ascendancySet := make(map[string]bool)
	for _, char := range chars {
		if !baseClasses[char.Ascendancy] {
			ascendancySet[char.Ascendancy] = true
		}
	}
	return len(ascendancySet) >= n
}
