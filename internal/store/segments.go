package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/getflaggy/flaggy/internal/models"
)

var ErrSegmentInUse = errors.New("segment is referenced by one or more rules")

func (s *SQLiteStore) CreateSegment(segment *models.Segment) error {
	now := time.Now().UTC()
	segment.CreatedAt = now
	segment.UpdatedAt = now

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO segments (key, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?)`,
		segment.Key, segment.Description, segment.CreatedAt, segment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert segment: %w", err)
	}

	for i := range segment.Conditions {
		c := &segment.Conditions[i]
		c.CreatedAt = now
		res, err := tx.Exec(
			`INSERT INTO segment_conditions (segment_key, attribute, operator, value, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			segment.Key, c.Attribute, c.Operator, string(c.Value), c.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert segment condition: %w", err)
		}
		cID, _ := res.LastInsertId()
		c.ID = cID
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetSegment(key string) (*models.Segment, error) {
	seg := &models.Segment{}
	err := s.db.QueryRow(
		`SELECT key, description, created_at, updated_at FROM segments WHERE key = ?`, key,
	).Scan(&seg.Key, &seg.Description, &seg.CreatedAt, &seg.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get segment: %w", err)
	}

	conditions, err := s.getSegmentConditions(key)
	if err != nil {
		return nil, err
	}
	seg.Conditions = conditions
	return seg, nil
}

func (s *SQLiteStore) ListSegments() ([]models.Segment, error) {
	rows, err := s.db.Query(
		`SELECT key, description, created_at, updated_at FROM segments ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}
	defer rows.Close()

	var segments []models.Segment
	for rows.Next() {
		var seg models.Segment
		if err := rows.Scan(&seg.Key, &seg.Description, &seg.CreatedAt, &seg.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan segment: %w", err)
		}
		segments = append(segments, seg)
	}
	return segments, rows.Err()
}

func (s *SQLiteStore) UpdateSegment(key string, req *models.UpdateSegmentRequest) (*models.Segment, error) {
	seg, err := s.GetSegment(key)
	if err != nil {
		return nil, err
	}
	if seg == nil {
		return nil, nil
	}

	if req.Description != nil {
		seg.Description = *req.Description
	}
	seg.UpdatedAt = time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE segments SET description = ?, updated_at = ? WHERE key = ?`,
		seg.Description, seg.UpdatedAt, key,
	)
	if err != nil {
		return nil, fmt.Errorf("update segment: %w", err)
	}

	if req.Conditions != nil {
		if _, err := tx.Exec(`DELETE FROM segment_conditions WHERE segment_key = ?`, key); err != nil {
			return nil, fmt.Errorf("delete segment conditions: %w", err)
		}

		seg.Conditions = make([]models.Condition, len(req.Conditions))
		for i, c := range req.Conditions {
			res, err := tx.Exec(
				`INSERT INTO segment_conditions (segment_key, attribute, operator, value, created_at)
				 VALUES (?, ?, ?, ?, ?)`,
				key, c.Attribute, c.Operator, string(c.Value), seg.UpdatedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("insert segment condition: %w", err)
			}
			cID, _ := res.LastInsertId()
			seg.Conditions[i] = models.Condition{
				ID:        cID,
				Attribute: c.Attribute,
				Operator:  c.Operator,
				Value:     c.Value,
				CreatedAt: seg.UpdatedAt,
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return seg, nil
}

func (s *SQLiteStore) DeleteSegment(key string) error {
	// Check if segment is referenced by any rule
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM rule_segments WHERE segment_key = ?`, key,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("check segment usage: %w", err)
	}
	if count > 0 {
		return ErrSegmentInUse
	}

	res, err := s.db.Exec(`DELETE FROM segments WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete segment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("segment not found")
	}
	return nil
}

func (s *SQLiteStore) getSegmentConditions(segmentKey string) ([]models.Condition, error) {
	rows, err := s.db.Query(
		`SELECT id, attribute, operator, value, created_at
		 FROM segment_conditions WHERE segment_key = ? ORDER BY id`, segmentKey,
	)
	if err != nil {
		return nil, fmt.Errorf("get segment conditions: %w", err)
	}
	defer rows.Close()

	var conditions []models.Condition
	for rows.Next() {
		var c models.Condition
		var val string
		if err := rows.Scan(&c.ID, &c.Attribute, &c.Operator, &val, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan segment condition: %w", err)
		}
		c.Value = json.RawMessage(val)
		conditions = append(conditions, c)
	}
	return conditions, rows.Err()
}
