package main

import (
	"go-racer/racer"
	"log"
)

func main() {
	racer, err := racer.NewRacerModel()

	if err != nil {
		log.Fatal(err)
	}

	if err := racer.Run(); err != nil {
		log.Fatal(err)
	}
}
