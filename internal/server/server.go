package server

import (
	"context"
	"github.com/CvitoyBamp/gopher/internal/config"
	"github.com/CvitoyBamp/gopher/internal/database"
	"log"
	"net/http"
)

type BackendServer struct {
	Server *http.Server
	DB     *database.Postgres
}

var cfg = config.Config

func DefaultBackendServer(createTables bool) (*BackendServer, error) {
	dbConfig, err := database.PGConfigParser(cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	dbInstance := database.NewPostgresInstance(context.Background(), dbConfig)

	if createTables {
		errTables := dbInstance.CreateTables()
		if errTables != nil {
			return nil, errTables
		}
	}

	return &BackendServer{
		Server: &http.Server{},
		DB:     dbInstance,
	}, nil
}

func StartService() {
	srv, err := DefaultBackendServer(true)
	if err != nil {
		log.Fatalf("can't server, error: %v", err)
	}
	log.Fatal(http.ListenAndServe(cfg.RunAddress, srv.router()))
}
