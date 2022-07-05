package service

import (
	"errors"
	"github.com/NikWaltz/metrics-collector/internal/storage"
	"github.com/NikWaltz/metrics-collector/model"
	"strconv"
	"strings"
)

type Storage interface {
	SaveGauge(string, model.Gauge)
	SaveCounter(string, model.Counter)
	GetGauge(string) (model.Gauge, bool)
	GetCounter(string) (model.Counter, bool)
	GetAll() storage.Storage
}

type service struct {
	storage Storage
}

func New(storage Storage) *service {
	return &service{storage: storage}
}

type TypeError struct {
}

func (e *TypeError) Error() string {
	return "wrong metric type"
}

func (s *service) GetGauge(name string) (model.Gauge, error) {
	if value, ok := s.storage.GetGauge(name); ok {
		return value, nil
	} else {
		return 0, errors.New("metric not exist")
	}

}

func (s *service) GetCounter(name string) (model.Counter, error) {
	if value, ok := s.storage.GetCounter(name); ok {
		return value, nil
	} else {
		return 0, errors.New("metric not exist")
	}
}

func (s *service) GetAll() storage.Storage {
	return s.storage.GetAll()
}

func (s *service) Update(metricType string, metricName string, metricValue string) error {
	switch strings.ToLower(metricType) {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return err
		}
		s.storage.SaveGauge(metricName, model.Gauge(value))
		return nil
	case "counter":
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
