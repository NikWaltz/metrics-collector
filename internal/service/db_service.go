package service

import (
	"context"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NikWaltz/metrics-collector/model"
)

type dbService struct {
	storage model.Storage
	pool    *pgxpool.Pool
}

func NewDBService(storage *model.Storage, dsn string) *dbService {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Println(err)
	}
	errMigrate := m.Steps(2)
	if errMigrate != nil {
		log.Println(errMigrate)
	}
	pool, errPool := pgxpool.New(context.Background(), dsn)
	if errPool != nil {
		log.Println(errPool)
	}
	return &dbService{storage: *storage, pool: pool}
}

func (s *dbService) Ping(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		log.Printf("Unable to acquire a database connection: %v\n", err)
	}
	defer conn.Release()
	return conn.Ping(ctx)
}

func (s *dbService) GetGauge(ctx context.Context, id string) (model.Gauge, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		log.Printf("Unable to acquire a database connection: %v\n", err)
		return 0, err
	}
	defer conn.Release()
	var value model.Gauge
	errRow := conn.QueryRow(ctx, `SELECT value FROM gauges WHERE id=$1;`, id).Scan(&value)
	if errRow != nil {
		log.Println(errRow)
		return 0, errRow
	}
	return value, nil
}

func (s *dbService) GetCounter(ctx context.Context, id string) (model.Counter, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		log.Printf("Unable to acquire a database connection: %v\n", err)
		return 0, err
	}
	defer conn.Release()
	var value model.Counter
	errRow := conn.QueryRow(ctx, `SELECT value FROM counters WHERE id=$1;`, id).Scan(&value)
	if errRow != nil {
		log.Println(errRow)
		return 0, errRow
	}
	return value, nil
}

func (s *dbService) GetStorage(ctx context.Context) model.Storage {
	return s.storage
}

func (s *dbService) Update(ctx context.Context, metricType string, metricName string, metricValue string) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		log.Printf("Unable to acquire a database connection: %v\n", err)
		return err
	}
	defer conn.Release()
	switch strings.ToLower(metricType) {
	case model.GaugeType:
		_, errExec := conn.Query(ctx,
			`INSERT INTO gauges(id, value) VALUES($1,$2) ON CONFLICT (id) DO UPDATE SET value=$2`, metricName, metricValue)
		if errExec != nil {
			log.Println(errExec)
			return errExec
		}
		return nil
	case model.CounterType:
		_, errExec := conn.Exec(ctx,
			`INSERT INTO counters(id, value) VALUES($1,$2) ON CONFLICT (id) DO UPDATE SET value=EXCLUDED.value + counters.value`, metricName, metricValue)
		if errExec != nil {
			log.Println(errExec)
			return errExec
		}
		return nil
	default:
		return &TypeError{}
	}
}

func (s *dbService) Close() {
	s.pool.Close()
}
