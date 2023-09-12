package config

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v9"
	"log"
)

var Config struct {
	RunAddress           string `env:"RUN_ADDRESS" envDefault:":8080"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	SecretToken          string `env:"TOKEN"`
}

func init() {

	flag.StringVar(&Config.RunAddress, "a", ":8080", "Address and port of service.")
	flag.StringVar(&Config.DatabaseURI, "d", "postgres://postgres:m7Pcsm6O0zFbMfms@db.ggtptvatdppjeksvebvk.supabase.co:5432/postgres", "DSN of PG database.")
	flag.StringVar(&Config.AccrualSystemAddress, "r", ":8081", "Charging system address.")
	flag.StringVar(&Config.SecretToken, "t", "secret", "Secret token for jwt.")
	flag.Parse()

	flag.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "" {
			log.Fatal(fmt.Sprintf("Flag \"-%s\" not set! It's necessary! Check --help flag.", f.Name))
		}
	})

	if err := env.Parse(&Config); err != nil {
		log.Fatalf("It's not possible to initialise environment variables, error: %v", err)
	}
}
