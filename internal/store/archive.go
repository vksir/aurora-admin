package store

import (
	"context"
	"encoding/json"
	"time"

	"aurora-admin/internal/entity/dstety"
	"github.com/google/uuid"
)

type Archive struct {
	ID                 string                `json:"id"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
	Remark             string                `json:"remark"`
	ClusterName        string                `json:"cluster_name"`
	ClusterPassword    string                `json:"cluster_password"`
	ClusterDescription string                `json:"cluster_description"`
	MaxPlayers         int                   `json:"max_players"`
	Pvp                bool                  `json:"pvp"`
	World              []dstety.DontStarveWorld `json:"world"`
	Mods               []*Mod                `json:"mods,omitempty"`
}

func (db *DB) CreateArchive(ctx context.Context, a *Archive) (*Archive, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	a.CreatedAt = time.Now()
	a.UpdatedAt = a.CreatedAt

	worldJSON, err := json.Marshal(a.World)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(ctx,
		`INSERT INTO dont_starve_archives (id, created_at, updated_at, remark, cluster_name, cluster_password, cluster_description, max_players, pvp, world)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.CreatedAt, a.UpdatedAt, a.Remark, a.ClusterName, a.ClusterPassword, a.ClusterDescription, a.MaxPlayers, a.Pvp, string(worldJSON))
	if err != nil {
		return nil, err
	}

	// Insert mod associations
	for _, m := range a.Mods {
		_, err = db.Exec(ctx, `INSERT OR IGNORE INTO archive_mods (archive_id, mod_id) VALUES (?, ?)`, a.ID, m.ID)
		if err != nil {
			return nil, err
		}
	}
	return db.GetArchive(ctx, a.ID)
}

func (db *DB) GetArchive(ctx context.Context, id string) (*Archive, error) {
	a := &Archive{}
	var worldJSON string
	err := db.QueryRow(ctx,
		`SELECT id, created_at, updated_at, remark, cluster_name, cluster_password, cluster_description, max_players, pvp, world
		 FROM dont_starve_archives WHERE id = ?`, id).
		Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt, &a.Remark, &a.ClusterName, &a.ClusterPassword, &a.ClusterDescription, &a.MaxPlayers, &a.Pvp, &worldJSON)
	if err != nil {
		return nil, err
	}
	if worldJSON != "" {
		json.Unmarshal([]byte(worldJSON), &a.World)
	}
	// Load mods
	a.Mods, err = db.getArchiveMods(ctx, id)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (db *DB) ListArchives(ctx context.Context) ([]*Archive, error) {
	rows, err := db.Query(ctx,
		`SELECT id, created_at, updated_at, remark, cluster_name, cluster_password, cluster_description, max_players, pvp, world
		 FROM dont_starve_archives ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var archives []*Archive
	for rows.Next() {
		a := &Archive{}
		var worldJSON string
		err := rows.Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt, &a.Remark, &a.ClusterName, &a.ClusterPassword, &a.ClusterDescription, &a.MaxPlayers, &a.Pvp, &worldJSON)
		if err != nil {
			return nil, err
		}
		if worldJSON != "" {
			json.Unmarshal([]byte(worldJSON), &a.World)
		}
		// Load mods
		a.Mods, err = db.getArchiveMods(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		archives = append(archives, a)
	}
	return archives, rows.Err()
}

func (db *DB) UpdateArchive(ctx context.Context, a *Archive) (*Archive, error) {
	a.UpdatedAt = time.Now()

	worldJSON, err := json.Marshal(a.World)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(ctx,
		`UPDATE dont_starve_archives SET updated_at=?, remark=?, cluster_name=?, cluster_password=?, cluster_description=?, max_players=?, pvp=?, world=?
		 WHERE id=?`,
		a.UpdatedAt, a.Remark, a.ClusterName, a.ClusterPassword, a.ClusterDescription, a.MaxPlayers, a.Pvp, string(worldJSON), a.ID)
	if err != nil {
		return nil, err
	}

	// Replace mod associations
	if a.Mods != nil {
		_, err = db.Exec(ctx, `DELETE FROM archive_mods WHERE archive_id = ?`, a.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range a.Mods {
			_, err = db.Exec(ctx, `INSERT OR IGNORE INTO archive_mods (archive_id, mod_id) VALUES (?, ?)`, a.ID, m.ID)
			if err != nil {
				return nil, err
			}
		}
	}
	return db.GetArchive(ctx, a.ID)
}

func (db *DB) DeleteArchive(ctx context.Context, id string) error {
	_, err := db.Exec(ctx, `DELETE FROM dont_starve_archives WHERE id = ?`, id)
	return err
}

func (db *DB) getArchiveMods(ctx context.Context, archiveID string) ([]*Mod, error) {
	rows, err := db.Query(ctx,
		`SELECT m.id, m.created_at, m.updated_at, m.workshop_id, m.name, m.description, m.image, m.time_created, m.time_updated, m.raw_config
		 FROM dont_starve_mods m
		 JOIN archive_mods am ON m.id = am.mod_id
		 WHERE am.archive_id = ?`, archiveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMods(rows)
}
