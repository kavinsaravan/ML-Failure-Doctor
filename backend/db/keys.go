package db

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type APIKey struct {
	ID        string     `json:"id"`
	OwnerID   string     `json:"owner_id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

func KeyHash(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func (db *DB) IssueAPIKey(name, owner string) (APIKey, string, error) {
	id, err := randomID()
	if err != nil {
		return APIKey{}, "", err
	}
	secret, err := randomID()
	if err != nil {
		return APIKey{}, "", err
	}
	if owner == "" {
		owner = id
	}
	token := "cl_" + secret
	_, err = db.Exec("INSERT INTO api_keys(id,owner_id,name,key_hash) VALUES(?,?,?,?)", id, owner, name, KeyHash(token))
	return APIKey{ID: id, OwnerID: owner, Name: name}, token, err
}
func (db *DB) ResolveAPIKey(token string) (string, error) {
	var owner string
	err := db.QueryRow("SELECT owner_id FROM api_keys WHERE key_hash=? AND revoked_at IS NULL", KeyHash(token)).Scan(&owner)
	return owner, err
}
func (db *DB) HasAPIKeys() (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM api_keys").Scan(&count)
	return count > 0, err
}
func (db *DB) GetAPIKeys() ([]APIKey, error) {
	rows, err := db.Query("SELECT id,owner_id,name,created_at,revoked_at FROM api_keys ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []APIKey{}
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.ID, &key.OwnerID, &key.Name, &key.CreatedAt, &key.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}
func (db *DB) RevokeAPIKey(id string) (bool, error) {
	result, err := db.Exec("UPDATE api_keys SET revoked_at=CURRENT_TIMESTAMP WHERE id=? AND revoked_at IS NULL", id)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count > 0, err
}
