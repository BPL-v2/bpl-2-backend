package scoring

import (
	"bpl/repository"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	_ "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
)

var db *gorm.DB
var enumQueries = []string{
	`CREATE TYPE bpl2.scoring_method AS ENUM ('PRESENCE', 'POINTS_FROM_VALUE', 'RANKED_TIME', 'RANKED_VALUE', 'RANKED_REVERSE', 'RANKED_COMPLETION_TIME', 'BONUS_PER_COMPLETION')`,
	`CREATE TYPE bpl2.objective_type AS ENUM ('ITEM', 'PLAYER', 'SUBMISSION')`,
	`CREATE TYPE bpl2.operator AS ENUM ('EQ', 'NEQ', 'GT', 'GTE', 'LT', 'LTE', 'IN', 'NOT_IN', 'MATCHES', 'CONTAINS', 'CONTAINS_ALL', 'CONTAINS_MATCH', 'CONTAINS_ALL_MATCHES')`,
	`CREATE TYPE bpl2.scoring_preset_type AS ENUM ('OBJECTIVE', 'CATEGORY')`,
	`CREATE TYPE bpl2.item_field AS ENUM ('BASE_TYPE', 'NAME', 'TYPE_LINE', 'RARITY', 'ILVL', 'FRAME_TYPE', 'TALISMAN_TIER', 'ENCHANT_MODS', 'EXPLICIT_MODS', 'IMPLICIT_MODS', 'CRAFTED_MODS', 'FRACTURED_MODS', 'SIX_LINK')`,
	`CREATE TYPE bpl2.number_field AS ENUM ('STACK_SIZE', 'PLAYER_LEVEL', 'PLAYER_XP', 'SUBMISSION_VALUE')`,
	`CREATE TYPE bpl2.approval_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED')`,
}

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("postgres", "17.2-alpine", []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "DATABASE_NAME=postgres"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	resource.Expire(600) // Tell docker to hard kill the container in 10 minutes
	sqlInfo := fmt.Sprintf(
		"host=localhost port=%s user=postgres password=postgres dbname=postgres sslmode=disable search_path=bpl2",
		resource.GetPort("5432/tcp"))

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		db, err = gorm.Open(postgres.Open(sqlInfo), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   "bpl2.",
				SingularTable: false,
			},
			Logger: logger.Default.LogMode(logger.Silent),
		})

		if err != nil {
			return err
		}
		db.Exec(`CREATE SCHEMA IF NOT EXISTS bpl2`)
		for _, query := range enumQueries {
			x := db.Exec(query)
			if x.Error != nil {
				if strings.Contains(x.Error.Error(), "already exists") {
					continue
				}
			}
		}
		return db.AutoMigrate(
			&repository.ScoringCategory{},
			&repository.Objective{},
			&repository.Condition{},
			&repository.Event{},
			&repository.Team{},
			&repository.User{},
			&repository.TeamUser{},
			&repository.StashChange{},
			&repository.ObjectiveMatch{},
			&repository.Submission{},
			&repository.ClientCredentials{},
			&repository.Signup{},
			&repository.Oauth{},
			&repository.KafkaConsumer{},
		)

	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	// as of go1.15 testing.M returns the exit code of m.Run(), so it is safe to use defer here
	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}

	}()
	m.Run()
}

func TearDown() {
	db.Exec("DELETE FROM bpl2.objective_matches")
	db.Exec("DELETE FROM bpl2.objectives")
	db.Exec("DELETE FROM bpl2.conditions")
	db.Exec("DELETE FROM bpl2.scoring_categories")
	db.Exec("DELETE FROM bpl2.events")
	db.Exec("DELETE FROM bpl2.teams")
	db.Exec("DELETE FROM bpl2.users")
	db.Exec("DELETE FROM bpl2.team_users")
	db.Exec("DELETE FROM bpl2.stash_changes")
}

func SetUp() *repository.Event {
	event := &repository.Event{
		Name:                 "event1",
		ScoringCategoryID:    1,
		MaxSize:              10,
		IsCurrent:            true,
		ApplicationStartTime: time.Now(),
		EventStartTime:       time.Now(),
		EventEndTime:         time.Now(),
		Teams: []*repository.Team{
			{
				Name: "team1",
				Users: []*repository.User{
					{
						DisplayName: "user1",
					},
					{
						DisplayName: "user2",
					},
				},
				AllowedClasses: []string{},
			},
			{
				Name: "team2",
				Users: []*repository.User{
					{
						DisplayName: "user3",
					},
					{
						DisplayName: "user4",
					},
				},
				AllowedClasses: []string{},
			},
		},
		ScoringCategory: &repository.ScoringCategory{
			Name: "category1",
		},
	}
	err := db.Create(event).Error
	if err != nil {
		log.Fatalf("Error creating event: %v", err)
	}
	return event
}

func TestAggregateMatchesEarliestFresh(t *testing.T) {
	// this tests that an objective that has the aggregation type of EARLIEST_FRESH_ITEM will only be counted as finished if the item
	// stays with the team that found it until the end
	event := SetUp()
	defer TearDown()
	objective := &repository.Objective{
		Name:           "objective1",
		Aggregation:    repository.EARLIEST_FRESH_ITEM,
		RequiredAmount: 1,
		CategoryID:     event.ScoringCategory.ID,
		ObjectiveType:  repository.ITEM,
		NumberField:    repository.STACK_SIZE,
		SyncStatus:     repository.SyncStatusSynced,
	}
	err := db.Create(objective).Error
	if err != nil {
		t.Errorf("Error creating objective: %v", err)
	}
	now := time.Now()
	stashChanges := []*repository.StashChange{
		{
			StashID:      "stash1",
			NextChangeID: "1",
			EventID:      event.ID,
			Timestamp:    now,
		},
		{
			StashID:      "stash2",
			NextChangeID: "1",
			EventID:      event.ID,
			Timestamp:    now,
		},
		// stashes is found again in another change later
		{
			StashID:      "stash1",
			NextChangeID: "2",
			EventID:      event.ID,
			Timestamp:    now.Add(time.Hour),
		},
		{
			StashID:      "stash2",
			NextChangeID: "2",
			EventID:      event.ID,
			Timestamp:    now.Add(time.Hour),
		},
	}
	db.Create(stashChanges)

	objectiveMatches := []*repository.ObjectiveMatch{
		// objective match is found in the first stash in the first change
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now,
			Number:        1,
			UserID:        event.Teams[0].Users[0].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[0].ID,
		},
		// objective match is found in the second stash in the first change
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now,
			Number:        1,
			UserID:        event.Teams[1].Users[0].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[1].ID,
		},
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now.Add(time.Hour),
			Number:        1,
			UserID:        event.Teams[1].Users[0].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[2].ID,
		},
	}
	db.Create(objectiveMatches)

	matches, err := AggregateMatches(db, event, []*repository.Objective{objective})
	if err != nil {
		t.Errorf("Error in AggregateMatches: %v", err)
	}
	objMatches, ok := matches[objective.ID]
	assert.True(t, ok, "Objective should be found in matches")
	_, ok = objMatches[event.Teams[0].ID]
	assert.False(t, ok, "Team1 should not have a match since no match was found in the first stash change")
	team2Match, ok := objMatches[event.Teams[1].ID]
	assert.True(t, ok, "Team2 should have a match")
	assert.InDelta(t, now.Unix(), team2Match.Timestamp.Unix(), 1, "match should have the timestamp of the match when it was first found")
}

func TestAggregateMatchesEarliestFreshStashMixup(t *testing.T) {
	// this tests that trading an item to a player in the same team will not keep the finishing time and player of the match when it was first scored
	event := SetUp()
	// defer TearDown()
	objective := &repository.Objective{
		Name:           "objective1",
		Aggregation:    repository.EARLIEST_FRESH_ITEM,
		RequiredAmount: 1,
		CategoryID:     event.ScoringCategory.ID,
		ObjectiveType:  repository.ITEM,
		NumberField:    repository.STACK_SIZE,
		SyncStatus:     repository.SyncStatusSynced,
	}
	err := db.Create(objective).Error
	if err != nil {
		t.Errorf("Error creating objective: %v", err)
	}
	now := time.Now()
	stashChanges := []*repository.StashChange{
		{
			StashID:      "stash1",
			NextChangeID: "1",
			EventID:      event.ID,
			Timestamp:    now,
		},
		{
			StashID:      "stash1",
			NextChangeID: "1",
			EventID:      event.ID,
			Timestamp:    now.Add(time.Hour),
		},
		{
			StashID:      "stash2",
			NextChangeID: "2",
			EventID:      event.ID,
			Timestamp:    now.Add(time.Hour),
		},
	}
	db.Create(stashChanges)

	objectiveMatches := []*repository.ObjectiveMatch{
		// objective match is found in stash of user1 of team 1
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now,
			Number:        1,
			UserID:        event.Teams[0].Users[0].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[0].ID,
		},
		// objective match is found later only in stash of user2 of team 1
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now.Add(time.Hour),
			Number:        1,
			UserID:        event.Teams[0].Users[1].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[2].ID,
		},
	}
	db.Create(objectiveMatches)

	matches, err := AggregateMatches(db, event, []*repository.Objective{objective})
	if err != nil {
		t.Errorf("Error in AggregateMatches: %v", err)
	}
	objMatches, ok := matches[objective.ID]
	assert.True(t, ok, "Objective should be found in matches")
	match, ok := objMatches[event.Teams[0].ID]
	assert.True(t, ok, "Team1 still has a match")
	if !ok {
		return
	}

	assert.InDelta(t, now.Unix(), match.Timestamp.Unix(), 1, "match should have the timestamp of the match when it was first found")
	assert.Equal(t, event.Teams[0].Users[0].ID, match.UserID, "match should be for user1 of team1 since that was the first match found")
}

func TestAggregateMatchesEarliestFreshGetCorrectCompletionTime(t *testing.T) {
	// this tests that trading an item to a player in the same team will not keep the finishing time and player of the match when it was first scored
	event := SetUp()
	// defer TearDown()
	objective := &repository.Objective{
		Name:           "objective1",
		Aggregation:    repository.EARLIEST_FRESH_ITEM,
		RequiredAmount: 100,
		CategoryID:     event.ScoringCategory.ID,
		ObjectiveType:  repository.ITEM,
		NumberField:    repository.STACK_SIZE,
		SyncStatus:     repository.SyncStatusSynced,
	}
	err := db.Create(objective).Error
	if err != nil {
		t.Errorf("Error creating objective: %v", err)
	}
	now := time.Now()
	stashChanges := []*repository.StashChange{
		{
			StashID:      "stash1",
			NextChangeID: "1",
			EventID:      event.ID,
			Timestamp:    now,
		},
		{
			StashID:      "stash1",
			NextChangeID: "2",
			EventID:      event.ID,
			Timestamp:    now.Add(time.Hour),
		},
		{
			StashID:      "stash1",
			NextChangeID: "3",
			EventID:      event.ID,
			Timestamp:    now.Add(2 * time.Hour),
		},
	}
	db.Create(stashChanges)

	objectiveMatches := []*repository.ObjectiveMatch{
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now,
			Number:        20,
			UserID:        event.Teams[0].Users[0].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[0].ID,
		},
		// finished the objective in the second stash change
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now.Add(time.Hour),
			Number:        101,
			UserID:        event.Teams[0].Users[0].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[1].ID,
		},
		{
			ObjectiveID:   objective.ID,
			Timestamp:     now.Add(2 * time.Hour),
			Number:        200,
			UserID:        event.Teams[0].Users[0].ID,
			EventId:       event.ID,
			StashChangeID: &stashChanges[2].ID,
		},
	}
	db.Create(objectiveMatches)

	matches, err := AggregateMatches(db, event, []*repository.Objective{objective})
	if err != nil {
		t.Errorf("Error in AggregateMatches: %v", err)
	}
	objMatches, ok := matches[objective.ID]
	assert.True(t, ok, "Objective should be found in matches")
	match, ok := objMatches[event.Teams[0].ID]
	assert.True(t, ok, "Team1 has a match")
	if !ok {
		return
	}

	assert.InDelta(t, now.Add(time.Hour).Unix(), match.Timestamp.Unix(), 1, "match should have the timestamp of completion")
}
