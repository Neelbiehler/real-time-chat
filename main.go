package main

import (
	"log"
	"net/http"
)

func main() {
	hubInstance := newHub()
	go hubInstance.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hubInstance, w, r)
	})

	log.Println("Server started")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
