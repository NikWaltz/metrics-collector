package storage

import (
	"github.com/NikWaltz/metrics-collector/model"
)

type Storage struct {
	Gauges   map[string]model.Gauge
	Counters map[string]model.Counter
}

func (s *Storage) SaveGauge(name string, value model.Gauge) {
	s.Gauges[name] = value
}

func (s *Storage) SaveCounter(name string, value model.Counter) {
	s.Counters[name] = value
}

func (s *Storage) GetGauge(name string) (model.Gauge, bool) {
	value, ok := s.Gauges[name]
	return value, ok
}

func (s *Storage) GetCounter(name string) (model.Counter, bool) {
	value, ok := s.Counters[name]
	return value, ok
}

func (s *Storage) GetAll() Storage {
	return *s
}

func New() *Storage {
	return &Storage{Gauges: make(map[string]model.Gauge), Counters: make(map[string]model.Counter)}
}
