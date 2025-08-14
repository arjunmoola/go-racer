package main

import (
	"go-racer/racer"
	"log"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	racer, err := racer.NewRacerModel()

	if err != nil {
		log.Fatal(err)
	}

	if err := racer.Run(); err != nil {
		log.Fatal(err)
	}
}
