package model

type Storage struct {
	Gauges   map[string]Gauge
	Counters map[string]Counter
}

func (s *Storage) SaveGauge(name string, value Gauge) {
	s.Gauges[name] = value
}

func (s *Storage) SaveCounter(name string, value Counter) {
	s.Counters[name] = value
}

func (s *Storage) GetGauge(name string) (Gauge, bool) {
	value, ok := s.Gauges[name]
	return value, ok
}

func (s *Storage) GetCounter(name string) (Counter, bool) {
	value, ok := s.Counters[name]
	return value, ok
}

func NewStorage() *Storage {
	return &Storage{Gauges: make(map[string]Gauge), Counters: make(map[string]Counter)}
}
