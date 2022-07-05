package api

import (
	"fmt"
	"github.com/NikWaltz/metrics-collector/internal/service"
	"github.com/NikWaltz/metrics-collector/internal/storage"
	"github.com/NikWaltz/metrics-collector/model"
	"github.com/go-chi/chi/v5"
	"html/template"
	"log"
	"net/http"
	"strings"
)

type Collector interface {
	Update(string, string, string) error
	GetGauge(string) (model.Gauge, error)
	GetCounter(string) (model.Counter, error)
	GetAll() storage.Storage
}

type api struct {
	r       chi.Router
	service Collector
}

func New(service Collector) *api {
	r := chi.NewRouter()

	return &api{service: service, r: r}
}

func (a *api) updateHandle(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")
	err := a.service.Update(metricType, metricName, metricValue)
	if err != nil {
		if _, ok := err.(*service.TypeError); ok {
			w.WriteHeader(http.StatusNotImplemented)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (a *api) valueHandle(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	switch strings.ToLower(metricType) {
	case "gauge":
		if value, err := a.service.GetGauge(metricName); err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("%v", value)))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case "counter":
		if value, err := a.service.GetCounter(metricName); err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("%d", value)))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) handle(w http.ResponseWriter, r *http.Request) {
	data := a.service.GetAll()
	data.Counters["PollCount"] = 24
	htmlTemplate := `{{range $index, $element := .Gauges}}{{$index}} {{printf "%f" $element}}
{{end}}{{range $index, $element := .Counters}}{{$index}} {{printf "%d" $element}}
{{end}}`
	tmpl, err := template.New("metrics").Parse(htmlTemplate)
	if err != nil {
		log.Println(err)
	}
	tmpl.Execute(w, &data)
}

func (a *api) Run() error {
	a.r.Post("/update/{type}/{name}/{value}", a.updateHandle)
	a.r.Get("/value/{type}/{name}", a.valueHandle)
	a.r.Get("/", a.handle)
	return http.ListenAndServe("localhost:8080", a.r)
}
