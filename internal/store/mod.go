package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Mod struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	WorkshopID  string    `json:"workshop_id"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Image       string    `json:"image,omitempty"`
	TimeCreated int64     `json:"time_created,omitempty"`
	TimeUpdated int64     `json:"time_updated,omitempty"`
	RawConfig   string    `json:"raw_config,omitempty"`
	Archives    []*Archive `json:"archives,omitempty"`
}

func (db *DB) CreateMod(ctx context.Context, m *Mod) (*Mod, error) {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	m.CreatedAt = time.Now()
	m.UpdatedAt = m.CreatedAt

	_, err := db.Exec(ctx,
		`INSERT INTO dont_starve_mods (id, created_at, updated_at, workshop_id, name, description, image, time_created, time_updated, raw_config)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.CreatedAt, m.UpdatedAt, m.WorkshopID, m.Name, m.Description, m.Image, m.TimeCreated, m.TimeUpdated, m.RawConfig)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (db *DB) GetMod(ctx context.Context, id string) (*Mod, error) {
	m := &Mod{}
	err := db.QueryRow(ctx,
		`SELECT id, created_at, updated_at, workshop_id, name, description, image, time_created, time_updated, raw_config
		 FROM dont_starve_mods WHERE id = ?`, id).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt, &m.WorkshopID, &m.Name, &m.Description, &m.Image, &m.TimeCreated, &m.TimeUpdated, &m.RawConfig)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (db *DB) GetModByWorkshopID(ctx context.Context, workshopID string) (*Mod, error) {
	m := &Mod{}
	err := db.QueryRow(ctx,
		`SELECT id, created_at, updated_at, workshop_id, name, description, image, time_created, time_updated, raw_config
		 FROM dont_starve_mods WHERE workshop_id = ?`, workshopID).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt, &m.WorkshopID, &m.Name, &m.Description, &m.Image, &m.TimeCreated, &m.TimeUpdated, &m.RawConfig)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (db *DB) ListMods(ctx context.Context) ([]*Mod, error) {
	rows, err := db.Query(ctx,
		`SELECT id, created_at, updated_at, workshop_id, name, description, image, time_created, time_updated, raw_config
		 FROM dont_starve_mods ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMods(rows)
}

func (db *DB) UpdateMod(ctx context.Context, m *Mod) (*Mod, error) {
	m.UpdatedAt = time.Now()
	_, err := db.Exec(ctx,
		`UPDATE dont_starve_mods SET updated_at=?, workshop_id=?, name=?, description=?, image=?, time_created=?, time_updated=?, raw_config=?
		 WHERE id=?`,
		m.UpdatedAt, m.WorkshopID, m.Name, m.Description, m.Image, m.TimeCreated, m.TimeUpdated, m.RawConfig, m.ID)
	if err != nil {
		return nil, err
	}
	return db.GetMod(ctx, m.ID)
}

func (db *DB) DeleteMod(ctx context.Context, id string) error {
	_, err := db.Exec(ctx, `DELETE FROM dont_starve_mods WHERE id = ?`, id)
	return err
}

func (db *DB) GetModWithArchives(ctx context.Context, id string) (*Mod, error) {
	m, err := db.GetMod(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx,
		`SELECT a.id, a.created_at, a.updated_at, a.remark, a.cluster_name, a.cluster_password, a.cluster_description, a.max_players, a.pvp, a.world
		 FROM dont_starve_archives a
		 JOIN archive_mods am ON a.id = am.archive_id
		 WHERE am.mod_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		a := &Archive{}
		var worldJSON string
		if err := rows.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt, &a.Remark, &a.ClusterName, &a.ClusterPassword, &a.ClusterDescription, &a.MaxPlayers, &a.Pvp, &worldJSON); err != nil {
			return nil, err
		}
		m.Archives = append(m.Archives, a)
	}
	return m, rows.Err()
}

func scanMods(rows *sql.Rows) ([]*Mod, error) {
	var mods []*Mod
	for rows.Next() {
		m := &Mod{}
		err := rows.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt, &m.WorkshopID, &m.Name, &m.Description, &m.Image, &m.TimeCreated, &m.TimeUpdated, &m.RawConfig)
		if err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}
	return nonNil(mods), rows.Err()
}
