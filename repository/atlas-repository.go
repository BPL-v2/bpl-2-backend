package repository

import (
	"bpl/config"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type AtlasRepository interface {
	CreateAtlasTree(userId int, eventId int, index int, nodes []int) error
	GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*AtlasTree, err error)
	GetLatestTreesForEvent(eventId int) (atlas []*AtlasTree, err error)
	GetAtlasesForEventAndUser(eventId int, userId int) (atlas []*AtlasTree, err error)
	// GetEventIdsWithAtlasHistory and GetActiveSecondsAtAtlasMilestones never return a nil slice without an error.
	GetEventIdsWithAtlasHistory() ([]int, error)
	GetActiveSecondsAtAtlasMilestones(eventId int, userIds []int, points []int, sessionGap time.Duration) ([]*AtlasMilestoneActiveTime, error)
}

const (
	// AtlasBonusNodeHash is the atlas node that grants AtlasBonusNodePoints additional points.
	// It must be subtracted from the amount of allocated nodes to get the points a player has,
	// see TrackedValueAtlasPoints in the parser.
	AtlasBonusNodeHash   = 65225
	AtlasBonusNodePoints = 20
	// minSnapshotsPerUserForHistory is how many atlas snapshots per player an event needs on average to
	// count as having atlas history. Events that only stored the final trees have ~2, others dozens.
	minSnapshotsPerUserForHistory = 10
)

type AtlasRepositoryImpl struct {
	DB *gorm.DB
}

func NewAtlasRepository() AtlasRepository {
	return &AtlasRepositoryImpl{DB: config.DatabaseConnection()}
}

type AtlasTree struct {
	UserID    int          `gorm:"not null"`
	EventID   int          `gorm:"not null"`
	Index     int          `gorm:"not null"`
	Nodes     PassiveNodes `gorm:"not null"`
	Timestamp time.Time    `gorm:"not null;default:current_timestamp"`

	User  *User  `gorm:"foreignKey:UserID"`
	Event *Event `gorm:"foreignKey:EventID"`
}

type PassiveNodes []int

func (a PassiveNodes) GetHash() [32]byte {
	buf := make([]byte, len(a)*4)
	for i, hash := range a {
		binary.LittleEndian.PutUint32(buf[i*4:], uint32(hash))
	}
	return sha256.Sum256(buf)
}
func (t *PassiveNodes) UnmarshalJSON(data []byte) error {
	var raw []int
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*t = PassiveNodes(raw)
	return nil
}

func (t PassiveNodes) MarshalJSON() ([]byte, error) {
	return json.Marshal([]int(t))
}

func (t *PassiveNodes) Scan(value any) error {
	var array pq.Int32Array
	if err := array.Scan(value); err != nil {
		return err
	}
	tree := make([]int, len(array))
	for i, v := range array {
		tree[i] = int(int32(v) + (1 << 15))
	}
	*t = PassiveNodes(tree)
	return nil
}

func (t PassiveNodes) Value() (driver.Value, error) {
	int32Array := make(pq.Int32Array, len(t))
	for i, v := range t {
		int32Array[i] = int32(v - (1 << 15))
	}
	return int32Array.Value()
}

func (r *AtlasRepositoryImpl) CreateAtlasTree(userId int, eventId int, index int, nodes []int) error {
	tree := &AtlasTree{
		UserID:    userId,
		EventID:   eventId,
		Nodes:     PassiveNodes(nodes),
		Index:     index,
		Timestamp: time.Now(),
	}
	return r.DB.Create(tree).Error
}

func (r *AtlasRepositoryImpl) GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*AtlasTree, err error) {
	query := `
        SELECT DISTINCT ON (a.user_id, a.index) a.* 
        FROM atlas_trees a
        JOIN team_users tu ON a.user_id = tu.user_id
        WHERE tu.team_id = ? AND a.event_id = ?
        ORDER BY a.user_id, a.index, a.timestamp DESC
    `
	err = r.DB.Raw(query, teamId, eventId).Scan(&atlas).Error
	if err != nil {
		return nil, err
	}
	return atlas, nil
}

func (r *AtlasRepositoryImpl) GetLatestTreesForEvent(eventId int) (atlas []*AtlasTree, err error) {
	query := `
		SELECT DISTINCT ON (a.user_id, a.index) a.*
		FROM atlas_trees a
		WHERE a.event_id = ?
		ORDER BY a.user_id, a.index, a.timestamp DESC
	`
	err = r.DB.Raw(query, eventId).Scan(&atlas).Error
	if err != nil {
		return nil, err
	}
	return atlas, nil
}

func (r *AtlasRepositoryImpl) GetAtlasesForEventAndUser(eventId int, userId int) (atlas []*AtlasTree, err error) {
	err = r.DB.Where("event_id = ? AND user_id = ?", eventId, userId).Order("timestamp DESC").Find(&atlas).Error
	if err != nil {
		return nil, err
	}
	return atlas, nil
}

// GetEventIdsWithAtlasHistory returns the events whose atlas trees were stored over time (snapshots
// whenever a player changed their tree) and not only once at the end of the event.
func (r *AtlasRepositoryImpl) GetEventIdsWithAtlasHistory() ([]int, error) {
	eventIds := make([]int, 0)
	err := r.DB.Raw(`
		SELECT event_id FROM atlas_trees
		GROUP BY event_id
		HAVING COUNT(*) >= ? * COUNT(DISTINCT user_id)
		ORDER BY event_id`, minSnapshotsPerUserForHistory).Scan(&eventIds).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching events with atlas history: %w", err)
	}
	if eventIds == nil { // Scan leaves the slice nil if there are no rows
		eventIds = []int{}
	}
	return eventIds, nil
}

// AtlasMilestoneActiveTime is the time a user was actively playing before first reaching an amount of
// atlas points.
type AtlasMilestoneActiveTime struct {
	UserId        int
	Points        int
	ActiveSeconds float64
}

// GetActiveSecondsAtAtlasMilestones returns, for every given user who reached one of the given amounts
// of atlas points in the event, how long they were actively playing until the first atlas snapshot with
// at least that many points. Activity rows closer together than sessionGap count as playing, longer
// gaps as breaks, and a session that is still running at the milestone counts up to that moment. So
// sleeping for ten hours after reaching a milestone does not count against the player. Users without
// activity in the event are left out.
// The reached time is when the points were allocated, not earned.
func (r *AtlasRepositoryImpl) GetActiveSecondsAtAtlasMilestones(
	eventId int, userIds []int, points []int, sessionGap time.Duration,
) ([]*AtlasMilestoneActiveTime, error) {
	if len(userIds) == 0 || len(points) == 0 {
		return []*AtlasMilestoneActiveTime{}, nil
	}
	pointValues := make([]string, len(points))
	for i, p := range points {
		pointValues[i] = fmt.Sprintf("%d", p)
	}
	// the database stores node hashes shifted by 2^15 to fit into smallints, see PassiveNodes.Scan
	query := fmt.Sprintf(`
		WITH ev AS (
			SELECT event_start_time, event_end_time FROM events WHERE id = @event_id
		), milestone AS (
			SELECT a.user_id, p.points, MIN(a.timestamp) AS reached_at
			FROM atlas_trees a
			CROSS JOIN (SELECT UNNEST(ARRAY[%s]::int[]) AS points) p
			WHERE a.event_id = @event_id AND a.user_id IN @user_ids
			  AND CARDINALITY(a.nodes) - CASE WHEN @bonus_node = ANY(a.nodes) THEN @bonus_points ELSE 0 END >= p.points
			GROUP BY a.user_id, p.points
		), pairs AS (
			SELECT a.user_id, a.time AS t0,
			       LEAD(a.time) OVER (PARTITION BY a.user_id ORDER BY a.time) AS t1
			FROM activity a, ev
			WHERE a.event_id = @event_id AND a.user_id IN @user_ids
			  AND a.time BETWEEN ev.event_start_time AND ev.event_end_time
		)
		SELECT m.user_id, m.points,
		       COALESCE(SUM(GREATEST(0, EXTRACT(EPOCH FROM (LEAST(p.t1, m.reached_at) - p.t0))))
		                FILTER (WHERE EXTRACT(EPOCH FROM (p.t1 - p.t0)) <= @gap_seconds), 0) AS active_seconds
		FROM milestone m
		JOIN pairs p ON p.user_id = m.user_id AND p.t0 < m.reached_at
		GROUP BY m.user_id, m.points
		ORDER BY m.user_id, m.points`, strings.Join(pointValues, ","))
	result := make([]*AtlasMilestoneActiveTime, 0)
	err := r.DB.Raw(query,
		map[string]any{
			"event_id":     eventId,
			"user_ids":     userIds,
			"bonus_node":   AtlasBonusNodeHash - (1 << 15),
			"bonus_points": AtlasBonusNodePoints,
			"gap_seconds":  sessionGap.Seconds(),
		}).Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("error fetching atlas milestone active times for event %d: %w", eventId, err)
	}
	if result == nil { // Scan leaves the slice nil if there are no rows
		result = []*AtlasMilestoneActiveTime{}
	}
	return result, nil
}
