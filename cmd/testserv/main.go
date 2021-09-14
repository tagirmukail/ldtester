package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()
	r.HandleFunc("/test1", func(w http.ResponseWriter, r *http.Request) {
		sleep := rand.Intn(3000010-1) + 1
		d := time.Microsecond * time.Duration(sleep)
		log.Println("/test1 start sleep:", d.String())
		<-time.After(d)
		log.Println("/test1 finish sleep:", d.String())
	})
	r.HandleFunc("/test2", func(w http.ResponseWriter, r *http.Request) {
		sleep := rand.Intn(3000010-1) + 1
		d := time.Microsecond * time.Duration(sleep)
		log.Println("/test2 start sleep:", d.String())
		<-time.After(d)
		log.Println("/test2 finish sleep:", d.String())
	})

	err := http.ListenAndServe(":8091", r)
	if err != nil {
		log.Fatal(err)
	}
}
