package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	tgToken := os.Getenv("TG_TOKEN")
	fmt.Println(tgToken)
}
