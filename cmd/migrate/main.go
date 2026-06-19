package main

import (
	"log"
	"oil/config"
	"oil/helper"
	"os"
)

const (
	argLength = 2
)

func main() {
	if len(os.Args) < argLength {
		log.Fatal("Migration direction (up/down) is required")
	}

	cfg := config.Get()

	switch os.Args[1] {
	case "up":
		if err := helper.Up(cfg); err != nil {
			log.Fatal(err)
		}
	case "down":
		if err := helper.Down(cfg); err != nil {
			log.Fatal(err)
		}
	case "drop":
		if err := helper.Drop(cfg); err != nil {
			log.Fatal(err)
		}
	case "step-up":
		if err := helper.StepUp(cfg); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Invalid direction. Use 'up', 'down', 'drop' or 'step-up'")
	}
}
