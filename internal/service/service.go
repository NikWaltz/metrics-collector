package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/NikWaltz/metrics-collector/model"
)

type service struct {
	storage model.Storage
}

func NewService(storage *model.Storage) *service {
	return &service{storage: *storage}
}

type TypeError struct {
}

func (e *TypeError) Error() string {
	return "wrong metric type"
}

func (s *service) GetGauge(ctx context.Context, name string) (model.Gauge, error) {
	if value, ok := s.storage.GetGauge(name); ok {
		return value, nil
	} else {
		return 0, errors.New("metric not exist")
	}

}

func (s *service) GetCounter(ctx context.Context, name string) (model.Counter, error) {
	if value, ok := s.storage.GetCounter(name); ok {
		return value, nil
	} else {
		return 0, errors.New("metric not exist")
	}
}

func (s *service) GetStorage(ctx context.Context) model.Storage {
	return s.storage
}

func (s *service) Update(ctx context.Context, metricType string, metricName string, metricValue string) error {
	switch strings.ToLower(metricType) {
	case model.GaugeType:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return err
		}
		s.storage.SaveGauge(metricName, model.Gauge(value))
		return nil
	case model.CounterType:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return err
		}
		newValue, _ := s.storage.GetCounter(metricName)
		newValue += model.Counter(value)
		s.storage.SaveCounter(metricName, newValue)
		return nil
	default:
		return &TypeError{}
	}
}

func (s *service) Ping(ctx context.Context) error {
	return nil
}
