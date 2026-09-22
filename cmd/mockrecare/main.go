// mockrecare is a stand-in for the Recare ingestion endpoint.
// Every 3rd request fails with 500 so the bridge's error/retry path
// (and what it does with the payload on failure) actually gets exercised.
package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	count := 0

	http.HandleFunc("/discharge-summaries", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		count++
		log.Printf("mockrecare: received %d bytes (request #%d)", len(body), count)

		if count%3 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"simulated downstream failure"}`))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status":"accepted"}`))
	})

	log.Println("mockrecare listening on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
