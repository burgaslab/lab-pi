package main

import (
	_ "fmt"
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("loooo"))
	})

	log.Fatal(http.ListenAndServe(":8008", nil))

}
