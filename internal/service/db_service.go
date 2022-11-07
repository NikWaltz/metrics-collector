package service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"

	"github.com/NikWaltz/metrics-collector/model"
)

type dbService struct {
	storage model.Storage
	dsn     string
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
	return &dbService{storage: *storage, dsn: dsn}
}

func (s *dbService) NewConnection(ctx context.Context) (*pgx.Conn, error) {
	connection, err := pgx.Connect(ctx, s.dsn)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	return connection, nil
}

func (s *dbService) Ping(ctx context.Context) error {
	connection, err := pgx.Connect(ctx, s.dsn)
	if err != nil {
		return err
	}
	return connection.Ping(ctx)
}

func (s *dbService) GetGauge(ctx context.Context, id string) (model.Gauge, error) {
	conn, err := s.NewConnection(ctx)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	defer conn.Close(ctx)
	var value model.Gauge
	errRow := conn.QueryRow(ctx, `SELECT value FROM gauges WHERE id=$1;`, id).Scan(&value)
	if errRow != nil {
		log.Println(errRow)
		return 0, errRow
	}
	return value, nil
}

func (s *dbService) GetCounter(ctx context.Context, id string) (model.Counter, error) {
	conn, err := s.NewConnection(ctx)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	defer conn.Close(ctx)
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
	conn, err := s.NewConnection(ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	defer conn.Close(ctx)
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
		var value int64
		var errExec error
		errRow := conn.QueryRow(ctx, `SELECT value FROM counters WHERE id=$1;`, metricName).Scan(&value)
		if errRow != nil {
			log.Println(err)
			_, errExec = conn.Exec(ctx,
				`INSERT INTO counters(id, value) VALUES($1,$2)`, metricName, metricValue)
		} else {
			delta, parseErr := strconv.ParseInt(metricValue, 10, 64)
			if parseErr != nil {
				log.Println(parseErr)
			}
			value += delta
			_, errExec = conn.Exec(ctx,
				`UPDATE counters SET value=$2 WHERE id=$1`, metricName, fmt.Sprintf("%d", value))
		}
		if errExec != nil {
			log.Println(errExec)
			return errExec
		}
		return nil
	default:
		return &TypeError{}
	}
}
