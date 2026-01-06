package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRegistry struct {
	db *pgxpool.Pool
	// We might cache the head for performance?
	// For now, simple direct DB access.
}

func NewPostgresRegistry(db *pgxpool.Pool) *PostgresRegistry {
	return &PostgresRegistry{db: db}
}

// Ensure interface compliance
var _ Store = (*PostgresRegistry)(nil)

func (r *PostgresRegistry) AppendBundle(ctx context.Context, b Bundle) (Bundle, error) {
	if b.Created.IsZero() {
		b.Created = time.Now().UTC()
	}

	// Get current head for PrevHash
	head, err := r.Head(ctx)
	if err != nil {
		return Bundle{}, err
	}
	if head != nil {
		b.PrevHash = head.Hash
		b.Version = head.Version + 1
	} else {
		b.Version = 1
	}

	h, err := hashBundle(b)
	if err != nil {
		return Bundle{}, err
	}
	b.Hash = h

	content, err := json.Marshal(b)
	if err != nil {
		return Bundle{}, err
	}

	// active=true is default
	_, err = r.db.Exec(ctx, `
		INSERT INTO policies (id, version, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, b.Hash, b.Version, content, b.Created)

	if err != nil {
		return Bundle{}, fmt.Errorf("failed to insert bundle: %w", err)
	}

	return b, nil
}

func (r *PostgresRegistry) Head(ctx context.Context) (*Bundle, error) {
	var content []byte
	err := r.db.QueryRow(ctx, `
		SELECT content FROM policies 
		WHERE active = true
		ORDER BY version DESC 
		LIMIT 1
	`).Scan(&content)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get head: %w", err)
	}

	var b Bundle
	if err := json.Unmarshal(content, &b); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bundle: %w", err)
	}
	return &b, nil
}

func (r *PostgresRegistry) GetByVersion(ctx context.Context, version int) (*Bundle, error) {
	var content []byte
	err := r.db.QueryRow(ctx, `
		SELECT content FROM policies 
		WHERE version = $1 
		ORDER BY active DESC 
		LIMIT 1
	`, version).Scan(&content)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bundle: %w", err)
	}

	var b Bundle
	if err := json.Unmarshal(content, &b); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bundle: %w", err)
	}
	return &b, nil
}

func (r *PostgresRegistry) GetByHash(ctx context.Context, hash string) (*Bundle, error) {
	var content []byte
	err := r.db.QueryRow(ctx, `
		SELECT content FROM policies WHERE id = $1
	`, hash).Scan(&content)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bundle: %w", err)
	}

	var b Bundle
	if err := json.Unmarshal(content, &b); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bundle: %w", err)
	}
	return &b, nil
}

func (r *PostgresRegistry) List(ctx context.Context, offset, limit int) ([]Bundle, int, error) {
	// Get total count
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM policies").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT content FROM policies 
		ORDER BY version ASC 
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var bundles []Bundle
	for rows.Next() {
		var content []byte
		if err := rows.Scan(&content); err != nil {
			return nil, 0, err
		}
		var b Bundle
		if err := json.Unmarshal(content, &b); err != nil {
			return nil, 0, err
		}
		bundles = append(bundles, b)
	}

	return bundles, total, nil
}

func (r *PostgresRegistry) ChainHashes(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, "SELECT id FROM policies ORDER BY version ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		hashes = append(hashes, id)
	}
	return hashes, nil
}

func (r *PostgresRegistry) VerifyChain(ctx context.Context) error {
	// For SQL, we probably don't need to re-verify everything constantly.
	// But to implement interface:
	// We could stream all bundles and verify.
	// For now, let's implement a lighter check or just assume consistency provided by AppendBundle logic?
	// Interface requires it.

	rows, err := r.db.Query(ctx, "SELECT content FROM policies ORDER BY version ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	var prevHash string
	i := 0
	for rows.Next() {
		var content []byte
		if err := rows.Scan(&content); err != nil {
			return err
		}
		var b Bundle
		if err := json.Unmarshal(content, &b); err != nil {
			return err
		}

		h, err := hashBundle(b)
		if err != nil {
			return err
		}
		if h != b.Hash {
			return fmt.Errorf("bundle hash mismatch at version %d", b.Version)
		}

		if i > 0 && b.PrevHash != prevHash {
			return fmt.Errorf("broken chain at version %d: prev %s != %s", b.Version, b.PrevHash, prevHash)
		}

		prevHash = b.Hash
		i++
	}
	return nil
}

func (r *PostgresRegistry) ActiveVersion(ctx context.Context) (int, error) {
	var v int
	err := r.db.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM policies WHERE active = true").Scan(&v)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (r *PostgresRegistry) Rollback(ctx context.Context, version int) error {
	// Mark all versions > version as inactive
	// Mark version as active (if it exists)
	// We need to ensure 'version' exists first.
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM policies WHERE version = $1)", version).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("rollback version %d not found", version)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Set active=false for newer
	_, err = tx.Exec(ctx, "UPDATE policies SET active = false WHERE version > $1", version)
	if err != nil {
		return err
	}
	// Set active=true for this version and older (in case they were previously inactive? rollback usually goes back)
	// Assuming linear history. If we rollback to V5, V5..V1 should be active?
	// The current logic implies Head() is max(version) where active=true.
	// So simply disabling > version is enough.
	// Re-enabling <= version?
	_, err = tx.Exec(ctx, "UPDATE policies SET active = true WHERE version <= $1", version)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Registry returns nil as we don't expose underlying in-memory legacy registry
func (r *PostgresRegistry) Registry() *Registry {
	return nil
}
