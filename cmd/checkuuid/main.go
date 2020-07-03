package main

import (
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"log"
)

func main() {
	id, err := machineid.ID()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(id)
}
