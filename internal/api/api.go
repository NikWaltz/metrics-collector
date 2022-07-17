package api

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/NikWaltz/metrics-collector/internal/service"
	"github.com/NikWaltz/metrics-collector/model"
)

type Collector interface {
	Update(string, string, string) error
	GetGauge(string) (model.Gauge, error)
	GetCounter(string) (model.Counter, error)
	GetStorage() model.Storage
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
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			log.Println(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (a *api) getValueHandle(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	switch strings.ToLower(metricType) {
	case "gauge":
		if value, err := a.service.GetGauge(metricName); err == nil {
			w.WriteHeader(http.StatusOK)
			_, errWr := w.Write([]byte(fmt.Sprintf("%v", value)))
			if errWr != nil {
				log.Println(errWr)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case "counter":
		if value, err := a.service.GetCounter(metricName); err == nil {
			w.WriteHeader(http.StatusOK)
			_, errWr := w.Write([]byte(fmt.Sprintf("%d", value)))
			if errWr != nil {
				log.Println(errWr)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) getMetricsHandle(w http.ResponseWriter, r *http.Request) {
	data := a.service.GetStorage()
	data.Counters["PollCount"] = 24
	htmlTemplate := `{{range $index, $element := .Gauges}}{{$index}} {{printf "%f" $element}}
{{end}}{{range $index, $element := .Counters}}{{$index}} {{printf "%d" $element}}
{{end}}`
	tmpl, err := template.New("metrics").Parse(htmlTemplate)
	if err != nil {
		log.Println(err)
	}
	errExec := tmpl.Execute(w, &data)
	if errExec != nil {
		log.Println(err)
	}
}

func (a *api) Run() error {
	a.r.Post("/update/{type}/{name}/{value}", a.updateHandle)
	a.r.Get("/value/{type}/{name}", a.getValueHandle)
	a.r.Get("/", a.getMetricsHandle)
	return http.ListenAndServe("localhost:8080", a.r)
}
