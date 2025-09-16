package main

import (
	"go-racer/internal/racer"
	"log"
	"os"
)

func main() {
	args := os.Args

	if len(args) > 2 {
		switch args[1] {
		case "add-test":
			if err := racer.RunAddTest(args[2:]); err != nil {
				log.Fatal(err)
			}
		}

		return
	}

	racerModel, err := racer.NewRacerModel()

	if err != nil {
		log.Fatal(err)
	}

	if err := racerModel.Run(); err != nil {
		log.Fatal(err)
	}
}
