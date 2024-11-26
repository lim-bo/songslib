package libmanager

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/lim-bo/songslib/models"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNoRows          error = errors.New(`no matching result`)
	ErrBadFilterParams error = errors.New(`bad query params for filtering`)
)

type Manager struct {
	mu   *sync.RWMutex
	pool *pgxpool.Pool
}

type DBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
}

func New(cfg DBConfig) *Manager {
	pool, err := pgxpool.Connect(context.Background(), "postgresql://"+cfg.Username+":"+cfg.Password+"@"+cfg.Host+":"+cfg.Port+"/"+cfg.DBName)
	if err != nil {
		log.Fatal("db connecting error: ", err)
	}
	return &Manager{
		mu:   &sync.RWMutex{},
		pool: pool,
	}
}

func (m *Manager) CreateSong(s *models.SongDetailed) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return errors.New("tx error: " + err.Error() + " rollback error: " + rbErr.Error())
		}
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO groups (name) VALUES ($1) ON CONFLICT (name) DO NOTHING;", s.Group)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return errors.New("tx error: " + err.Error() + " rollback error: " + rbErr.Error())
		}
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO songs (name, group, release_date, lyrics, link) VALUES ($1, (SELECT id FROM groups WHERE groups.name = $2) , $3, $4, $5);",
		s.Name, s.Group, s.ReleaseDate, s.Text, s.Link)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return errors.New("tx error: " + err.Error() + " rollback error: " + rbErr.Error())
		}
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return errors.New("tx error: " + err.Error() + " rollback error: " + rbErr.Error())
		}
	}
	return err
}

func (m *Manager) GetSong(s models.Song) (*models.SongDetailed, error) {
	var sd models.SongDetailed
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	row := m.pool.QueryRow(ctx, `SELECT name, group, release_date, lyrics, link FROM songs
						 LEFT JOIN groups ON songs.group = groups.id
						 WHERE name = $1 AND groups = $2;`, s.Name, s.Group)

	if err := row.Scan(&sd.Name, &sd.Group, &sd.ReleaseDate, &sd.Text, &sd.Link); err == pgx.ErrNoRows {
		return nil, ErrNoRows
	} else if err != nil {
		return nil, err
	}
	return &sd, nil
}

func (m *Manager) DeleteSong(s models.Song) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_, err := m.pool.Exec(ctx, `DELETE FROM songs WHERE name = $1, groups = (SELECT id FROM groups WHERE groups.name = $2 LIMIT 1);`, s.Name, s.Group)
	if err == pgx.ErrNoRows {
		return ErrNoRows
	}
	return err
}

func (m *Manager) GetSongsPage(page int, elemCountPerPage int, filter map[string]string) ([]*models.SongDetailed, error) {
	sdPage := make([]*models.SongDetailed, 0, elemCountPerPage)
	filterCondition := " "
	if len(filter) != 0 {
		filterCondition += `WHERE `
		for key, value := range filter {
			filterCondition += key + "=" + value + " AND"
		}
		filterCondition = strings.TrimSuffix(filterCondition, "AND")
		if strings.Contains(filterCondition, ";") {
			slog.Debug("incoming bad query params to filter: ", slog.String("result condition", filterCondition))
			return nil, ErrBadFilterParams
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	query := `SELECT name, group, release_date, lyrics, link FROM songs
	LEFT JOIN groups ON songs.group = groups.id` + filterCondition
	rows, err := m.pool.Query(ctx, query+`LIMIT $1 OFFSET $2;`, elemCountPerPage, page*elemCountPerPage)
	if err == pgx.ErrNoRows {
		return nil, ErrNoRows
	} else if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sd models.SongDetailed
		if err = rows.Scan(&sd.Name, &sd.Group, &sd.ReleaseDate, &sd.Text, &sd.Link); err != nil {
			return nil, err
		}
		sdPage = append(sdPage, &sd)
	}
	return sdPage, nil
}

func (m *Manager) UpdateSongData(newData *models.SongDetailed) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_, err := m.pool.Exec(ctx, `UPDATE songs SET release_date = $1, lyrics = $2, link = $3 WHERE
					 name = $4 AND group = (SELECT id FROM groups WHERE name = $5);`, newData.ReleaseDate, newData.Text, newData.Link,
		newData.Name, newData.Group)
	if err == pgx.ErrNoRows {
		return ErrNoRows
	}
	return err
}
