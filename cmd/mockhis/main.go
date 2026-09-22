// mockhis is a stand-in for a hospital's legacy HIS export endpoint.
// It just serves the seeded testdata so the bridge has something to pull.
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	data, err := os.ReadFile("testdata/discharge_records.json")
	if err != nil {
		log.Fatalf("mockhis: could not read testdata: %v", err)
	}

	http.HandleFunc("/records", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	log.Println("mockhis listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
