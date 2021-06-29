package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rddigital/web-test-api/messagebus"
)

func main() {
	r := InitRestRoutes()

	log.Println("Listening on:3333...")
	err := http.ListenAndServe("localhost:3333", r)
	if err != nil {
		log.Fatal(err)
	}
}

func InitRestRoutes() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
	})

	r.HandleFunc("/connect", messagebus.ConnectHandler).Methods(http.MethodPost)
	r.HandleFunc("/request/action/{requestID}/{method}/{path}", messagebus.MQTTRequestActionHandler).Methods(http.MethodPost)
	r.HandleFunc("/request/body/{requestID}/{method}/{path}", messagebus.MQTTBodyHandler).Methods(http.MethodPost)
	r.HandleFunc("/request/topic/{requestID}", messagebus.GetMQTTResponseTopicHandler).Methods(http.MethodGet)

	return r
}
