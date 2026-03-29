package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/getflaggy/flaggy/internal/models"
)

func (s *SQLiteStore) CreateAPIKey(key *models.APIKey, hashedKey string) error {
	_, err := s.db.Exec(
		`INSERT INTO api_keys (id, name, environment, prefix, hashed_key, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		key.ID, key.Name, key.Environment, key.Prefix, hashedKey, key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListAPIKeys() ([]models.APIKey, error) {
	rows, err := s.db.Query(
		`SELECT id, name, environment, prefix, revoked, created_at, last_used_at
		 FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		var lastUsed sql.NullTime
		if err := rows.Scan(&k.ID, &k.Name, &k.Environment, &k.Prefix,
			&k.Revoked, &k.CreatedAt, &lastUsed); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		if lastUsed.Valid {
			k.LastUsedAt = &lastUsed.Time
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *SQLiteStore) ValidateAPIKey(hashedKey string) (*models.APIKey, error) {
	var k models.APIKey
	var lastUsed sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, name, environment, prefix, revoked, created_at, last_used_at
		 FROM api_keys WHERE hashed_key = ? AND revoked = 0`, hashedKey,
	).Scan(&k.ID, &k.Name, &k.Environment, &k.Prefix, &k.Revoked, &k.CreatedAt, &lastUsed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("validate api key: %w", err)
	}
	if lastUsed.Valid {
		k.LastUsedAt = &lastUsed.Time
	}

	// Update last_used_at
	now := time.Now().UTC()
	s.db.Exec(`UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now, k.ID)

	return &k, nil
}

func (s *SQLiteStore) RevokeAPIKey(id string) error {
	res, err := s.db.Exec(`UPDATE api_keys SET revoked = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}
