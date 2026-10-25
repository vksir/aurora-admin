package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Admin struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	KleiID    string    `json:"klei_id"`
}

func (db *DB) CreateAdmin(ctx context.Context, a *Admin) (*Admin, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	a.CreatedAt = time.Now()
	a.UpdatedAt = a.CreatedAt

	_, err := db.Exec(ctx,
		`INSERT INTO dont_starve_admins (id, created_at, updated_at, name, klei_id) VALUES (?, ?, ?, ?, ?)`,
		a.ID, a.CreatedAt, a.UpdatedAt, a.Name, a.KleiID)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (db *DB) ListAdmins(ctx context.Context) ([]*Admin, error) {
	rows, err := db.Query(ctx, `SELECT id, created_at, updated_at, name, klei_id FROM dont_starve_admins ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []*Admin
	for rows.Next() {
		a := &Admin{}
		if err := rows.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt, &a.Name, &a.KleiID); err != nil {
			return nil, err
		}
		admins = append(admins, a)
	}
	return admins, rows.Err()
}

func (db *DB) DeleteAdmin(ctx context.Context, id string) error {
	_, err := db.Exec(ctx, `DELETE FROM dont_starve_admins WHERE id = ?`, id)
	return err
}
