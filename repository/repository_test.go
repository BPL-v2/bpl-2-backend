package repository

import (
	"fmt"
	"log"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}
	resource, err := pool.Run("postgres", "17.2-alpine", []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "DATABASE_NAME=postgres"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	err = resource.Expire(600)
	if err != nil {
		log.Fatalf("Could not set resource expiration: %s", err)
	}
	sqlInfo := fmt.Sprintf(
		"host=localhost port=%s user=postgres password=postgres dbname=postgres sslmode=disable search_path=bpl2",
		resource.GetPort("5432/tcp"))

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
		err = db.AutoMigrate(
			&Event{},
			&Objective{},
			&Condition{},
			&Team{},
			&User{},
			&TeamUser{},
			&StashChange{},
			&ObjectiveMatch{},
			&Submission{},
			&ClientCredentials{},
			&Signup{},
			&Oauth{},
			&KafkaConsumer{},
			&ScoringPreset{},
			&ObjectiveScoringPreset{},
			&Character{},
			&CharacterPob{},
			&ChangeId{},
		)
		if err != nil {
			fmt.Println("Error in AutoMigrate: ", err)
		}
		return err
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}()
	m.Run()
}

func tearDown() {
	db.Exec("DELETE FROM bpl2.objective_scoring_presets")
	db.Exec("DELETE FROM bpl2.scoring_presets")
	db.Exec("DELETE FROM bpl2.submissions")
	db.Exec("DELETE FROM bpl2.objective_matches")
	db.Exec("DELETE FROM bpl2.objectives")
	db.Exec("DELETE FROM bpl2.signups")
	db.Exec("DELETE FROM bpl2.stash_changes")
	db.Exec("DELETE FROM bpl2.change_ids")
	db.Exec("DELETE FROM bpl2.character_pobs")
	db.Exec("DELETE FROM bpl2.characters")
	db.Exec("DELETE FROM bpl2.team_users")
	db.Exec("DELETE FROM bpl2.teams")
	db.Exec("DELETE FROM bpl2.oauths")
	db.Exec("DELETE FROM bpl2.users")
	db.Exec("DELETE FROM bpl2.client_credentials")
	db.Exec("DELETE FROM bpl2.kafka_consumers")
	db.Exec("DELETE FROM bpl2.events")
}

func createTestEvent() *Event {
	event := &Event{
		Name:                 "test-event",
		MaxSize:              10,
		IsCurrent:            true,
		GameVersion:          PoE2,
		ApplicationStartTime: time.Now().Add(-time.Hour),
		ApplicationEndTime:   time.Now().Add(time.Hour),
		EventStartTime:       time.Now(),
		EventEndTime:         time.Now().Add(24 * time.Hour),
	}
	db.Create(event)
	return event
}

func createTestUsers(n int) []*User {
	users := make([]*User, n)
	for i := range n {
		users[i] = &User{DisplayName: fmt.Sprintf("user%d", i+1)}
	}
	db.Create(&users)
	return users
}

func createTestTeamsWithUsers(event *Event) ([]*Team, []*User) {
	users := createTestUsers(4)
	teams := []*Team{
		{Name: "team1", Abbreviation: "T1", EventId: event.Id, Color: "#ff0000", AllowedClasses: []string{}},
		{Name: "team2", Abbreviation: "T2", EventId: event.Id, Color: "#0000ff", AllowedClasses: []string{}},
	}
	db.Create(&teams)
	db.Create(&TeamUser{TeamId: teams[0].Id, UserId: users[0].Id})
	db.Create(&TeamUser{TeamId: teams[0].Id, UserId: users[1].Id})
	db.Create(&TeamUser{TeamId: teams[1].Id, UserId: users[2].Id})
	db.Create(&TeamUser{TeamId: teams[1].Id, UserId: users[3].Id})
	return teams, users
}

// ==================== UserRepository Tests ====================

func TestUserRepository_GetUserById(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	user := &User{DisplayName: "testuser", Permissions: Permissions{PermissionAdmin}}
	db.Create(user)

	found, err := repo.GetUserById(user.Id)
	require.NoError(t, err)
	assert.Equal(t, "testuser", found.DisplayName)
	assert.Contains(t, found.Permissions, PermissionAdmin)
}

func TestUserRepository_GetUserById_NotFound(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}

	_, err := repo.GetUserById(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_GetUsersByIds(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	users := createTestUsers(3)

	found, err := repo.GetUsersByIds([]int{users[0].Id, users[2].Id})
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestUserRepository_SaveUser(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	user := &User{DisplayName: "newuser"}

	saved, err := repo.SaveUser(user)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)
	assert.Equal(t, "newuser", saved.DisplayName)

	// Update
	saved.DisplayName = "updated"
	saved, err = repo.SaveUser(saved)
	require.NoError(t, err)
	assert.Equal(t, "updated", saved.DisplayName)
}

func TestUserRepository_GetAllUsers(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	createTestUsers(3)

	users, err := repo.GetAllUsers()
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestUserRepository_GetStreamersForEvent(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	_ = teams

	// Add twitch oauth for user 0
	db.Create(&Oauth{
		UserId:        users[0].Id,
		Provider:      ProviderTwitch,
		AccessToken:   "tok",
		Expiry:        time.Now().Add(time.Hour),
		RefreshExpiry: time.Now().Add(24 * time.Hour),
		Name:          "twitchuser",
		AccountId:     "twitch123",
	})

	streamers, err := repo.GetStreamersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, streamers, 1)
	assert.Equal(t, users[0].Id, streamers[0].UserId)
	assert.Equal(t, "twitch123", streamers[0].TwitchId)
}

func TestUserRepository_GetUsersForEvent(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	event := createTestEvent()
	_, users := createTestTeamsWithUsers(event)

	// Add PoE oauth for first two users
	for i := 0; i < 2; i++ {
		db.Create(&Oauth{
			UserId:        users[i].Id,
			Provider:      ProviderPoE,
			AccessToken:   fmt.Sprintf("token%d", i),
			Expiry:        time.Now().Add(time.Hour),
			RefreshExpiry: time.Now().Add(24 * time.Hour),
			Name:          fmt.Sprintf("poeuser%d", i),
			AccountId:     fmt.Sprintf("poe%d", i),
		})
	}

	result, err := repo.GetUsersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestUser_HasOneOfPermissions(t *testing.T) {
	user := &User{Permissions: Permissions{PermissionAdmin, PermissionManager}}

	assert.True(t, user.HasOneOfPermissions(PermissionAdmin))
	assert.True(t, user.HasOneOfPermissions(PermissionManager))
	assert.False(t, user.HasOneOfPermissions(PermissionSubmissionJudge))
	assert.True(t, user.HasOneOfPermissions(PermissionSubmissionJudge, PermissionAdmin))
}

func TestUser_HasPoEName(t *testing.T) {
	user := &User{
		OauthAccounts: []*Oauth{
			{Provider: ProviderPoE, Name: "TestAccount#1234"},
		},
	}
	assert.True(t, user.HasPoEName("testaccount"))
	assert.True(t, user.HasPoEName("TestAccount#5678"))
	assert.False(t, user.HasPoEName("otheraccount"))
}

// ==================== EventRepository Tests ====================

func TestEventRepository_GetCurrentEvent(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}

	event := createTestEvent()

	found, err := repo.GetCurrentEvent()
	require.NoError(t, err)
	assert.Equal(t, event.Id, found.Id)
	assert.Equal(t, "test-event", found.Name)
	assert.True(t, found.IsCurrent)
}

func TestEventRepository_GetCurrentEvent_NotFound(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}

	_, err := repo.GetCurrentEvent()
	assert.Error(t, err)
}

func TestEventRepository_GetEventById(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	event := createTestEvent()

	found, err := repo.GetEventById(event.Id)
	require.NoError(t, err)
	assert.Equal(t, event.Name, found.Name)
}

func TestEventRepository_GetEventById_WithPreloads(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	event := createTestEvent()
	db.Create(&Team{Name: "t1", Abbreviation: "T1", EventId: event.Id, Color: "#fff", AllowedClasses: []string{}})

	found, err := repo.GetEventById(event.Id, "Teams")
	require.NoError(t, err)
	assert.Len(t, found.Teams, 1)
	assert.Equal(t, "t1", found.Teams[0].Name)
}

func TestEventRepository_InvalidateCurrentEvent(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	createTestEvent()

	err := repo.InvalidateCurrentEvent()
	require.NoError(t, err)

	_, err = repo.GetCurrentEvent()
	assert.Error(t, err, "no current event should exist after invalidation")
}

func TestEventRepository_FindAll(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}

	for i := range 3 {
		e := &Event{
			Name:                 fmt.Sprintf("event%d", i),
			MaxSize:              10,
			GameVersion:          PoE2,
			ApplicationStartTime: time.Now(),
			ApplicationEndTime:   time.Now(),
			EventStartTime:       time.Now(),
			EventEndTime:         time.Now(),
		}
		db.Create(e)
	}

	events, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestEventRepository_Delete(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	event := createTestEvent()

	err := repo.Delete(event)
	require.NoError(t, err)

	_, err = repo.GetEventById(event.Id)
	assert.Error(t, err)
}

// ==================== TeamRepository Tests ====================

func TestTeamRepository_GetTeamById(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	team := &Team{Name: "myteam", Abbreviation: "MT", EventId: event.Id, Color: "#000", AllowedClasses: []string{}}
	db.Create(team)

	found, err := repo.GetTeamById(team.Id)
	require.NoError(t, err)
	assert.Equal(t, "myteam", found.Name)
}

func TestTeamRepository_GetTeamsForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	createTestTeamsWithUsers(event)

	teams, err := repo.GetTeamsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, teams, 2)
}

func TestTeamRepository_Save(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	team := &Team{Name: "newteam", Abbreviation: "NT", EventId: event.Id, Color: "#123", AllowedClasses: []string{"Warrior"}}

	saved, err := repo.Save(team)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)

	saved.Name = "renamed"
	saved, err = repo.Save(saved)
	require.NoError(t, err)
	assert.Equal(t, "renamed", saved.Name)
}

func TestTeamRepository_Delete(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, _ := createTestTeamsWithUsers(event)

	err := repo.Delete(teams[0].Id)
	require.NoError(t, err)

	_, err = repo.GetTeamById(teams[0].Id)
	assert.Error(t, err)

	// Team users should also be deleted
	var count int64
	db.Model(&TeamUser{}).Where("team_id = ?", teams[0].Id).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestTeamRepository_GetTeamUsersForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	createTestTeamsWithUsers(event)

	teamUsers, err := repo.GetTeamUsersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, teamUsers, 4)
}

func TestTeamRepository_AddUsersToTeams(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	team := &Team{Name: "t1", Abbreviation: "T1", EventId: event.Id, Color: "#000", AllowedClasses: []string{}}
	db.Create(team)
	users := createTestUsers(2)

	err := repo.AddUsersToTeams([]*TeamUser{
		{TeamId: team.Id, UserId: users[0].Id},
		{TeamId: team.Id, UserId: users[1].Id},
	})
	require.NoError(t, err)

	teamUsers, err := repo.GetTeamUsersForTeam(team.Id)
	require.NoError(t, err)
	assert.Len(t, teamUsers, 2)
}

func TestTeamRepository_RemoveUserForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	_, users := createTestTeamsWithUsers(event)

	err := repo.RemoveUserForEvent(users[0].Id, event.Id)
	require.NoError(t, err)

	teamUsers, err := repo.GetTeamUsersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, teamUsers, 3)
}

func TestTeamRepository_GetTeamForUser(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)

	tu, err := repo.GetTeamForUser(event.Id, users[0].Id)
	require.NoError(t, err)
	assert.Equal(t, teams[0].Id, tu.TeamId)
}

func TestTeamRepository_GetTeamLeadsForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	_ = teams

	// Make user0 a team lead
	db.Model(&TeamUser{}).Where("user_id = ? AND team_id = ?", users[0].Id, teams[0].Id).Update("is_team_lead", true)

	leads, err := repo.GetTeamLeadsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, leads, 1)
	assert.Equal(t, users[0].Id, leads[0].UserId)
}

func TestTeamRepository_GetNumbersOfPastEventsParticipatedByUsers(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}

	// Create two past events
	past1 := &Event{Name: "past1", MaxSize: 10, GameVersion: PoE2, IsCurrent: false,
		ApplicationStartTime: time.Now(), ApplicationEndTime: time.Now(), EventStartTime: time.Now(), EventEndTime: time.Now()}
	past2 := &Event{Name: "past2", MaxSize: 10, GameVersion: PoE2, IsCurrent: false,
		ApplicationStartTime: time.Now(), ApplicationEndTime: time.Now(), EventStartTime: time.Now(), EventEndTime: time.Now()}
	db.Create(past1)
	db.Create(past2)

	users := createTestUsers(2)
	t1 := &Team{Name: "pt1", Abbreviation: "PT1", EventId: past1.Id, Color: "#000", AllowedClasses: []string{}}
	t2 := &Team{Name: "pt2", Abbreviation: "PT2", EventId: past2.Id, Color: "#000", AllowedClasses: []string{}}
	db.Create(t1)
	db.Create(t2)
	db.Create(&TeamUser{TeamId: t1.Id, UserId: users[0].Id})
	db.Create(&TeamUser{TeamId: t2.Id, UserId: users[0].Id})
	db.Create(&TeamUser{TeamId: t1.Id, UserId: users[1].Id})

	result, err := repo.GetNumbersOfPastEventsParticipatedByUsers([]int{users[0].Id, users[1].Id})
	require.NoError(t, err)
	assert.Equal(t, 2, result[users[0].Id])
	assert.Equal(t, 1, result[users[1].Id])
}

// ==================== ObjectiveRepository Tests ====================

func TestObjectiveRepository_SaveAndGetObjective(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{
		Name:           "test-objective",
		EventId:        event.Id,
		ObjectiveType:  ObjectiveTypeItem,
		NumberField:    NumberFieldStackSize,
		Aggregation:    AggregationTypeEarliest,
		RequiredAmount: 5,
		SyncStatus:     SyncStatusSynced,
	}
	saved, err := repo.SaveObjective(obj)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)
	assert.Equal(t, SyncStatusDesynced, saved.SyncStatus, "SaveObjective should set status to DESYNCED")

	found, err := repo.GetObjectiveById(saved.Id)
	require.NoError(t, err)
	assert.Equal(t, "test-objective", found.Name)
	assert.Equal(t, 5, found.RequiredAmount)
}

func TestObjectiveRepository_DeleteObjective(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{Name: "todelete", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, NumberField: NumberFieldStackSize, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	err := repo.DeleteObjective(obj.Id)
	require.NoError(t, err)

	_, err = repo.GetObjectiveById(obj.Id)
	assert.Error(t, err)
}

func TestObjectiveRepository_GetObjectivesByEventIdFlat(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	root := &Objective{Name: "root", EventId: event.Id, ObjectiveType: ObjectiveTypeCategory, NumberField: NumberFieldFinishedObjectives, Aggregation: AggregationTypeNone, SyncStatus: SyncStatusDesynced}
	db.Create(root)
	child := &Objective{Name: "child", EventId: event.Id, ParentId: &root.Id, ObjectiveType: ObjectiveTypeItem, NumberField: NumberFieldStackSize, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(child)

	objectives, err := repo.GetObjectivesByEventIdFlat(event.Id)
	require.NoError(t, err)
	assert.Len(t, objectives, 2)
}

func TestObjectiveRepository_GetObjectivesByEventId_Tree(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	root := &Objective{Name: "root", EventId: event.Id, ObjectiveType: ObjectiveTypeCategory, NumberField: NumberFieldFinishedObjectives, Aggregation: AggregationTypeNone, SyncStatus: SyncStatusDesynced}
	db.Create(root)
	child1 := &Objective{Name: "child1", EventId: event.Id, ParentId: &root.Id, ObjectiveType: ObjectiveTypeItem, NumberField: NumberFieldStackSize, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	child2 := &Objective{Name: "child2", EventId: event.Id, ParentId: &root.Id, ObjectiveType: ObjectiveTypePlayer, NumberField: NumberFieldPlayerLevel, Aggregation: AggregationTypeLatest, SyncStatus: SyncStatusDesynced}
	db.Create(child1)
	db.Create(child2)

	tree, err := repo.GetObjectivesByEventId(event.Id)
	require.NoError(t, err)
	assert.Equal(t, "root", tree.Name)
	assert.Len(t, tree.Children, 2)
}

func TestObjectiveRepository_SyncStatusLifecycle(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{Name: "synctest", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, NumberField: NumberFieldStackSize, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	err := repo.StartSync([]int{obj.Id})
	require.NoError(t, err)
	found, _ := repo.GetObjectiveById(obj.Id)
	assert.Equal(t, SyncStatusSyncing, found.SyncStatus)

	err = repo.FinishSync([]int{obj.Id})
	require.NoError(t, err)
	found, _ = repo.GetObjectiveById(obj.Id)
	assert.Equal(t, SyncStatusSynced, found.SyncStatus)
}

func TestObjectiveRepository_AssociateScoringPresets(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{Name: "obj", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, NumberField: NumberFieldStackSize, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)
	preset1 := &ScoringPreset{EventId: event.Id, Name: "p1", Description: "d1", Points: ExtendingNumberSlice{10}, ScoringMethod: PRESENCE}
	preset2 := &ScoringPreset{EventId: event.Id, Name: "p2", Description: "d2", Points: ExtendingNumberSlice{5}, ScoringMethod: RANKED_TIME}
	db.Create(preset1)
	db.Create(preset2)

	err := repo.AssociateScoringPresets(obj.Id, []int{preset1.Id, preset2.Id})
	require.NoError(t, err)

	found, err := repo.GetObjectiveById(obj.Id, "ScoringPresets")
	require.NoError(t, err)
	assert.Len(t, found.ScoringPresets, 2)

	// Re-associate with only one preset
	err = repo.AssociateScoringPresets(obj.Id, []int{preset1.Id})
	require.NoError(t, err)
	found, err = repo.GetObjectiveById(obj.Id, "ScoringPresets")
	require.NoError(t, err)
	assert.Len(t, found.ScoringPresets, 1)
}

func TestObjective_FlatMap(t *testing.T) {
	root := &Objective{Name: "root", Children: []*Objective{
		{Name: "child1", Children: []*Objective{
			{Name: "grandchild"},
		}},
		{Name: "child2"},
	}}
	flat := root.FlatMap()
	assert.Len(t, flat, 4)
}

// ==================== SignupRepository Tests ====================

func TestSignupRepository_SaveAndGetSignup(t *testing.T) {
	defer tearDown()
	repo := &SignupRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	signup := &Signup{
		EventId:          event.Id,
		UserId:           users[0].Id,
		Timestamp:        time.Now(),
		ExpectedPlayTime: 40,
		NeedsHelp:        true,
	}
	saved, err := repo.SaveSignup(signup)
	require.NoError(t, err)
	assert.Equal(t, event.Id, saved.EventId)

	found, err := repo.GetSignupForUser(users[0].Id, event.Id)
	require.NoError(t, err)
	assert.Equal(t, 40, found.ExpectedPlayTime)
	assert.True(t, found.NeedsHelp)
}

func TestSignupRepository_RemoveSignupForUser(t *testing.T) {
	defer tearDown()
	repo := &SignupRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	signup := &Signup{EventId: event.Id, UserId: users[0].Id, Timestamp: time.Now(), ExpectedPlayTime: 20}
	db.Create(signup)

	err := repo.RemoveSignupForUser(users[0].Id, event.Id)
	require.NoError(t, err)

	_, err = repo.GetSignupForUser(users[0].Id, event.Id)
	assert.Error(t, err)
}

func TestSignupRepository_GetSignupsForEvent(t *testing.T) {
	defer tearDown()
	repo := &SignupRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(3)

	for i, u := range users {
		db.Create(&Signup{
			EventId:          event.Id,
			UserId:           u.Id,
			Timestamp:        time.Now().Add(time.Duration(i) * time.Minute),
			ExpectedPlayTime: 10 * (i + 1),
		})
	}

	signups, err := repo.GetSignupsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, signups, 3)
	// Should be ordered by timestamp ASC
	assert.True(t, signups[0].Timestamp.Before(signups[2].Timestamp) || signups[0].Timestamp.Equal(signups[2].Timestamp))
	// User should be preloaded
	assert.NotNil(t, signups[0].User)
}

// ==================== OauthRepository Tests ====================

func TestOauthRepository_SaveAndGet(t *testing.T) {
	defer tearDown()
	repo := &OauthRepositoryImpl{DB: db}
	users := createTestUsers(1)

	oauth := &Oauth{
		UserId:        users[0].Id,
		Provider:      ProviderDiscord,
		AccessToken:   "access123",
		RefreshToken:  "refresh123",
		Expiry:        time.Now().Add(time.Hour),
		RefreshExpiry: time.Now().Add(24 * time.Hour),
		Name:          "discorduser",
		AccountId:     "discord456",
	}
	saved, err := repo.SaveOauth(oauth)
	require.NoError(t, err)
	assert.Equal(t, "discorduser", saved.Name)

	found, err := repo.GetOauthByProviderAndAccountId(ProviderDiscord, "discord456")
	require.NoError(t, err)
	assert.Equal(t, users[0].Id, found.UserId)
	assert.NotNil(t, found.User, "User should be preloaded")
}

func TestOauthRepository_GetOauthByProviderAndAccountName(t *testing.T) {
	defer tearDown()
	repo := &OauthRepositoryImpl{DB: db}
	users := createTestUsers(1)

	db.Create(&Oauth{
		UserId:        users[0].Id,
		Provider:      ProviderPoE,
		AccessToken:   "tok",
		Expiry:        time.Now().Add(time.Hour),
		RefreshExpiry: time.Now().Add(24 * time.Hour),
		Name:          "poeaccount",
		AccountId:     "poe123",
	})

	found, err := repo.GetOauthByProviderAndAccountName(ProviderPoE, "poeaccount")
	require.NoError(t, err)
	assert.Equal(t, "poe123", found.AccountId)
}

func TestOauthRepository_DeleteOauthsByUserIdAndProvider(t *testing.T) {
	defer tearDown()
	repo := &OauthRepositoryImpl{DB: db}
	users := createTestUsers(1)

	db.Create(&Oauth{UserId: users[0].Id, Provider: ProviderPoE, AccessToken: "t", Expiry: time.Now(), RefreshExpiry: time.Now(), Name: "n", AccountId: "a"})
	db.Create(&Oauth{UserId: users[0].Id, Provider: ProviderDiscord, AccessToken: "t", Expiry: time.Now(), RefreshExpiry: time.Now(), Name: "n", AccountId: "a"})

	err := repo.DeleteOauthsByUserIdAndProvider(users[0].Id, ProviderPoE)
	require.NoError(t, err)

	oauths, err := repo.GetAllOauths()
	require.NoError(t, err)
	assert.Len(t, oauths, 1)
	assert.Equal(t, ProviderDiscord, oauths[0].Provider)
}

// ==================== SubmissionRepository Tests ====================

func TestSubmissionRepository_SaveAndGet(t *testing.T) {
	defer tearDown()
	repo := &SubmissionRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "subobj", EventId: event.Id, ObjectiveType: ObjectiveTypeSubmission, NumberField: NumberFieldSubmissionValue, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	sub := &Submission{
		ObjectiveId:    obj.Id,
		Timestamp:      time.Now(),
		Number:         42,
		UserId:         users[0].Id,
		TeamId:         teams[0].Id,
		Proof:          "https://example.com/proof.png",
		Comment:        "look at this",
		ApprovalStatus: PENDING,
	}
	saved, err := repo.SaveSubmission(sub)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)

	found, err := repo.GetSubmissionById(saved.Id)
	require.NoError(t, err)
	assert.Equal(t, 42, found.Number)
	assert.Equal(t, PENDING, found.ApprovalStatus)
	assert.NotNil(t, found.Objective)
}

func TestSubmissionRepository_GetSubmissionsForEvent(t *testing.T) {
	defer tearDown()
	repo := &SubmissionRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	event.Teams = teams // Must be set for TeamIds() to work
	obj := &Objective{Name: "subobj", EventId: event.Id, ObjectiveType: ObjectiveTypeSubmission, NumberField: NumberFieldSubmissionValue, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	for i := 0; i < 3; i++ {
		db.Create(&Submission{
			ObjectiveId:    obj.Id,
			Timestamp:      time.Now(),
			Number:         i + 1,
			UserId:         users[i%2].Id,
			TeamId:         teams[i%2].Id,
			Proof:          "proof",
			Comment:        "comment",
			ApprovalStatus: PENDING,
		})
	}

	subs, err := repo.GetSubmissionsForEvent(event)
	require.NoError(t, err)
	assert.Len(t, subs, 3)
}

func TestSubmissionRepository_DeleteSubmission(t *testing.T) {
	defer tearDown()
	repo := &SubmissionRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "subobj", EventId: event.Id, ObjectiveType: ObjectiveTypeSubmission, NumberField: NumberFieldSubmissionValue, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	sub := &Submission{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 1, UserId: users[0].Id, TeamId: teams[0].Id, Proof: "p", Comment: "c", ApprovalStatus: PENDING}
	db.Create(sub)

	err := repo.DeleteSubmission(sub.Id)
	require.NoError(t, err)

	_, err = repo.GetSubmissionById(sub.Id)
	assert.Error(t, err)
}

func TestSubmission_ToObjectiveMatch(t *testing.T) {
	sub := &Submission{
		ObjectiveId: 10,
		Timestamp:   time.Now(),
		Number:      5,
		UserId:      20,
		TeamId:      30,
	}
	match := sub.ToObjectiveMatch()
	assert.Equal(t, 10, match.ObjectiveId)
	assert.Equal(t, 5, match.Number)
	assert.Equal(t, 30, match.TeamId)
	assert.Equal(t, 20, *match.UserId)
}

// ==================== ScoringPresetRepository Tests ====================

func TestScoringPresetRepository_SaveAndGet(t *testing.T) {
	defer tearDown()
	repo := &ScoringPresetRepositoryImpl{DB: db}
	event := createTestEvent()

	preset := &ScoringPreset{
		EventId:       event.Id,
		Name:          "test-preset",
		Description:   "A test preset",
		Points:        ExtendingNumberSlice{10, 5, 3},
		ScoringMethod: RANKED_TIME,
	}
	saved, err := repo.SavePreset(preset)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)

	presets, err := repo.GetPresetsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, presets, 1)
	assert.Equal(t, "test-preset", presets[0].Name)
	assert.Equal(t, float64(10), presets[0].Points[0])
	assert.Equal(t, float64(5), presets[0].Points[1])
}

func TestScoringPresetRepository_DeletePreset(t *testing.T) {
	defer tearDown()
	repo := &ScoringPresetRepositoryImpl{DB: db}
	event := createTestEvent()

	preset := &ScoringPreset{EventId: event.Id, Name: "del", Description: "d", Points: ExtendingNumberSlice{1}, ScoringMethod: PRESENCE}
	db.Create(preset)

	err := repo.DeletePreset(preset.Id)
	require.NoError(t, err)

	presets, err := repo.GetPresetsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, presets, 0)
}

func TestExtendingNumberSlice_Get(t *testing.T) {
	s := ExtendingNumberSlice{10, 5, 3}
	assert.Equal(t, float64(10), s.Get(0))
	assert.Equal(t, float64(5), s.Get(1))
	assert.Equal(t, float64(3), s.Get(2))
	assert.Equal(t, float64(3), s.Get(100), "should return last element for out of bounds")

	empty := ExtendingNumberSlice{}
	assert.Equal(t, float64(0), empty.Get(0), "should return 0 for empty slice")
}

// ==================== ObjectiveMatchRepository Tests ====================

func TestObjectiveMatchRepository_SaveAndDeleteMatches(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveMatchRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "matchobj", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, NumberField: NumberFieldStackSize, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	matches := []*ObjectiveMatch{
		{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 1, TeamId: teams[0].Id, UserId: &users[0].Id},
		{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 2, TeamId: teams[1].Id, UserId: &users[2].Id},
	}
	err := repo.SaveMatches(matches)
	require.NoError(t, err)

	var count int64
	db.Model(&ObjectiveMatch{}).Where("objective_id = ?", obj.Id).Count(&count)
	assert.Equal(t, int64(2), count)

	err = repo.DeleteMatches([]int{obj.Id})
	require.NoError(t, err)

	db.Model(&ObjectiveMatch{}).Where("objective_id = ?", obj.Id).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestObjectiveMatchRepository_OverwriteMatches(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveMatchRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "overwrite", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, NumberField: NumberFieldStackSize, Aggregation: AggregationTypeEarliest, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	// Save initial matches
	db.Create(&ObjectiveMatch{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 1, TeamId: teams[0].Id, UserId: &users[0].Id})
	db.Create(&ObjectiveMatch{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 2, TeamId: teams[0].Id, UserId: &users[1].Id})

	// Overwrite with new matches
	newMatches := []*ObjectiveMatch{
		{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 99, TeamId: teams[1].Id, UserId: &users[2].Id},
	}
	err := repo.OverwriteMatches(newMatches, []int{obj.Id})
	require.NoError(t, err)

	var count int64
	db.Model(&ObjectiveMatch{}).Where("objective_id = ?", obj.Id).Count(&count)
	assert.Equal(t, int64(1), count)

	var match ObjectiveMatch
	db.Where("objective_id = ?", obj.Id).First(&match)
	assert.Equal(t, 99, match.Number)
}

// ==================== StashChangeRepository Tests ====================

func TestStashChangeRepository_CreateStashChangeIfNotExists(t *testing.T) {
	defer tearDown()
	repo := &StashChangeRepositoryImpl{DB: db}
	event := createTestEvent()
	now := time.Now().Truncate(time.Microsecond)

	sc := &StashChange{StashId: "stash-1", EventId: event.Id, Timestamp: now}
	created, err := repo.CreateStashChangeIfNotExists(sc)
	require.NoError(t, err)
	assert.NotZero(t, created.Id)

	// Creating again with same stash_id + event_id + timestamp should return existing
	sc2 := &StashChange{StashId: "stash-1", EventId: event.Id, Timestamp: now}
	existing, err := repo.CreateStashChangeIfNotExists(sc2)
	require.NoError(t, err)
	assert.Equal(t, created.Id, existing.Id)
}

func TestStashChangeRepository_GetLatestTimestamp(t *testing.T) {
	defer tearDown()
	repo := &StashChangeRepositoryImpl{DB: db}
	event := createTestEvent()
	now := time.Now()

	db.Create(&StashChange{StashId: "s1", EventId: event.Id, Timestamp: now.Add(-time.Hour)})
	db.Create(&StashChange{StashId: "s2", EventId: event.Id, Timestamp: now})

	latest, err := repo.GetLatestTimestamp(event.Id)
	require.NoError(t, err)
	assert.InDelta(t, now.Unix(), latest.Unix(), 1)
}

func TestStashChangeRepository_GetLatestTimestamp_NoRecords(t *testing.T) {
	defer tearDown()
	repo := &StashChangeRepositoryImpl{DB: db}

	latest, err := repo.GetLatestTimestamp(99999)
	require.NoError(t, err)
	assert.True(t, latest.IsZero())
}

// ==================== Conditions / Custom Types Tests ====================

func TestConditions_ScanValue(t *testing.T) {
	conditions := Conditions{
		{Field: BASE_TYPE, Operator: EQ, Value: "Mirror of Kalandra"},
		{Field: ILVL, Operator: GT, Value: "80"},
	}
	val, err := conditions.Value()
	require.NoError(t, err)

	var scanned Conditions
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Len(t, scanned, 2)
	assert.Equal(t, BASE_TYPE, scanned[0].Field)
	assert.Equal(t, GT, scanned[1].Operator)
}

func TestConditions_ScanNil(t *testing.T) {
	var c Conditions
	err := c.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, c)
}

func TestPermissions_ScanValue(t *testing.T) {
	perms := Permissions{PermissionAdmin, PermissionManager}
	val, err := perms.Value()
	require.NoError(t, err)

	var scanned Permissions
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Len(t, scanned, 2)
	assert.Equal(t, PermissionAdmin, scanned[0])
	assert.Equal(t, PermissionManager, scanned[1])
}

func TestExtraMap_ScanValue(t *testing.T) {
	extra := ExtraMap{"key1": "val1", "key2": "val2"}
	val, err := extra.Value()
	require.NoError(t, err)

	var scanned ExtraMap
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, "val1", scanned["key1"])
	assert.Equal(t, "val2", scanned["key2"])
}

func TestExtraMap_ScanNil(t *testing.T) {
	var e ExtraMap
	err := e.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, e)
}

// ==================== PoENameWithoutDiscriminator Tests ====================

func TestPoENameWithoutDiscriminator(t *testing.T) {
	name := "TestAccount#1234"
	assert.Equal(t, "testaccount", PoENameWithoutDiscriminator(&name))

	plain := "simpleaccount"
	assert.Equal(t, "simpleaccount", PoENameWithoutDiscriminator(&plain))

	assert.Equal(t, "", PoENameWithoutDiscriminator(nil))
}

// ==================== LoadUsersIntoEvent Tests ====================

func TestLoadUsersIntoEvent(t *testing.T) {
	defer tearDown()
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	event.Teams = teams

	err := LoadUsersIntoEvent(db, event)
	require.NoError(t, err)
	assert.Len(t, event.Teams[0].Users, 2)
	assert.Len(t, event.Teams[1].Users, 2)

	// Verify correct users in correct teams
	team0UserIds := []int{event.Teams[0].Users[0].Id, event.Teams[0].Users[1].Id}
	assert.Contains(t, team0UserIds, users[0].Id)
	assert.Contains(t, team0UserIds, users[1].Id)
}

// ==================== Event model method tests ====================

func TestEvent_TeamIds(t *testing.T) {
	event := &Event{
		Teams: []*Team{
			{Id: 1}, {Id: 5}, {Id: 10},
		},
	}
	ids := event.TeamIds()
	assert.Equal(t, []int{1, 5, 10}, ids)
}
