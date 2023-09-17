package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/CvitoyBamp/gopher/internal/config"
	"github.com/CvitoyBamp/gopher/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"log"
	"net/http"
	"time"
)

type AccrualService struct {
	Server *http.Server
	Client *http.Client
	DB     *database.Postgres
}

func createAccrualService() (*AccrualService, error) {
	dbConfig, err := database.PGConfigParser(cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	dbInstance := database.NewPostgresInstance(context.Background(), dbConfig)

	return &AccrualService{
		Server: &http.Server{},
		Client: &http.Client{},
		DB:     dbInstance,
	}, nil
}

var cfg = config.Config

func (as *AccrualService) accrualRouter() chi.Router {

	r := chi.NewRouter()

	r.Use(httprate.Limit(
		10,            // requests
		1*time.Second, // per duration
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "No more than N requests per minute allowed\n", http.StatusTooManyRequests)
		}),
	))

	r.Route("/api/orders", func(r chi.Router) {
		r.Post("/{number}", as.accrualCalc)
	})

	return r
}

func StartAccrualServer() {

	srv, errServ := createAccrualService()
	if errServ != nil {
		log.Fatal(errServ)
	}

	go func() {
		log.Fatal(http.ListenAndServe(cfg.AccrualSystemAddress, srv.accrualRouter()))
	}()

	srv.processOrders()

}

func (as *AccrualService) accrualCalc(w http.ResponseWriter, r *http.Request) {
	orderNumber := chi.URLParam(r, "number")

	order, errOrder := as.DB.GetAccrual(orderNumber)

	if errOrder != nil {
		if errOrder.Error() != "no rows in result set" {
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, errMarshal := json.Marshal(order)
	if errMarshal != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, errResp := fmt.Fprint(w, string(body))
	if errResp != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
}
