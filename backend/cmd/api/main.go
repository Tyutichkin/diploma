package main

import (
	"log"

	"planner-backend/internal/app"
	"planner-backend/internal/config"
)

func main() {
	cfg := config.MustLoad()

	a, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
