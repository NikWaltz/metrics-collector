package service

import (
	"errors"
	"github.com/NikWaltz/metrics-collector/internal/storage"
	"github.com/NikWaltz/metrics-collector/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdate(t *testing.T) {
	type fields struct {
		storage Storage
	}
	type args struct {
		metricType  string
		metricName  string
		metricValue string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "Update gauge metric",
			fields: fields{storage: storage.New()},
			args: args{
				metricType:  "gauge",
				metricName:  "TotalMemory",
				metricValue: "65.34",
			},
			wantErr: false,
		},
		{
			name:   "Update gauge metric with complex value",
			fields: fields{storage: storage.New()},
			args: args{
				metricType:  "gauge",
				metricName:  "TotalMemory",
				metricValue: "65 + 23i",
			},
			wantErr: true,
		},
		{
			name:   "Update counter metric",
			fields: fields{storage: storage.New()},
			args: args{
				metricType:  "counter",
				metricName:  "PollCounter",
				metricValue: "62",
			},
			wantErr: false,
		},
		{
			name:   "Update counter metric with float value",
			fields: fields{storage: storage.New()},
			args: args{
				metricType:  "counter",
				metricName:  "PollCounter",
				metricValue: "63.243",
			},
			wantErr: true,
		},
		{
			name:   "Update non-existence metric",
			fields: fields{storage: storage.New()},
			args: args{
				metricType:  "histogram",
				metricName:  "Total",
				metricValue: "63.243",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				storage: tt.fields.storage,
			}
			if tt.wantErr {
				assert.Error(t, s.Update(tt.args.metricType, tt.args.metricName, tt.args.metricValue))
			} else {
				assert.NoError(t, s.Update(tt.args.metricType, tt.args.metricName, tt.args.metricValue))
			}
		})
	}
}

func TestGetAll(t *testing.T) {
	st := &storage.Storage{
		Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
		Counters: map[string]model.Counter{"Counter": 5},
	}
	type fields struct {
		storage Storage
	}
	tests := []struct {
		name   string
		fields fields
		want   storage.Storage
	}{
		{
			name:   "Get storage",
			fields: fields{storage: st},
			want:   *st,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				storage: tt.fields.storage,
			}
			assert.Equalf(t, tt.want, s.GetAll(), "GetAll()")
		})
	}
}

func TestGetCounter(t *testing.T) {
	type fields struct {
		storage Storage
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    model.Counter
		wantErr error
	}{
		{
			name: "Get exist counter",
			fields: fields{storage: &storage.Storage{
				Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
				Counters: map[string]model.Counter{"PollCounter": 5},
			}},
			args:    args{name: "PollCounter"},
			want:    5,
			wantErr: nil,
		},
		{
			name: "Get non-existence counter",
			fields: fields{storage: &storage.Storage{
				Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
				Counters: map[string]model.Counter{"PollCounter": 5},
			}},
			args:    args{name: "SomeCounter"},
			want:    0,
			wantErr: errors.New("metric not exist"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				storage: tt.fields.storage,
			}
			got, err := s.GetCounter(tt.args.name)
			if tt.wantErr == nil {
				assert.NoError(t, tt.wantErr, err)
			} else {
				assert.Error(t, tt.wantErr, err)
			}
			assert.Equalf(t, tt.want, got, "GetCounter(%v)", tt.args.name)
		})
	}
}

func TestGetGauge(t *testing.T) {
	type fields struct {
		storage Storage
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    model.Gauge
		wantErr error
	}{
		{
			name: "Get exist gauge",
			fields: fields{storage: &storage.Storage{
				Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
				Counters: map[string]model.Counter{"PollCounter": 5},
			}},
			args:    args{name: "Alloc"},
			want:    43.53234,
			wantErr: nil,
		},
		{
			name: "Get non-existence gauge",
			fields: fields{storage: &storage.Storage{
				Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
				Counters: map[string]model.Counter{"PollCounter": 5},
			}},
			args:    args{name: "SomeGauge"},
			want:    0,
			wantErr: errors.New("metric not exist"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				storage: tt.fields.storage,
			}
			got, err := s.GetGauge(tt.args.name)
			if tt.wantErr == nil {
				assert.NoError(t, tt.wantErr, err)
			} else {
				assert.Error(t, tt.wantErr, err)
			}
			assert.Equalf(t, tt.want, got, "GetGauge(%v)", tt.args.name)
		})
	}
}
