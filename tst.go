package main

import (
	"fmt"
	"log"

	"github.com/andre487/data-migrators/providers/fatsecret"
	"github.com/andre487/data-migrators/utils/secrets"
)

func main() {
	var err error
	var keyData []byte
	if keyData, err = secrets.GetSecretFromFile("~/.tokens/fatsecret.json"); err != nil {
		log.Fatal(err)
	}

	fs, err := fatsecret.New(keyData)
	if err != nil {
		log.Fatal(err)
	}

	data, err := fs.GetTestData()
	fmt.Printf("ERROR: %+v\n", err)
	fmt.Printf("DATA: %+v\n", data)
}
