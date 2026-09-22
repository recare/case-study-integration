// discharge-bridge pulls discharge records from a hospital's legacy HIS
// export, maps them into the shape Recare expects, and forwards them on.
//
// NOTE: this is a first cut that's been running in prod for a while.
// It works, mostly. Take a look and see what you'd change before we
// scale it to more hospitals.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// ---- config -----------------------------------------------------------

type Config struct {
	HISEndpoint    string
	RecareEndpoint string
	HISAPIKey      string
	RecareAPIKey   string
}

var cfg Config // global, loaded once in main

func loadConfig(path string) Config {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read config: %v", err)
	}

	values := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		values[key] = val
	}

	c := Config{
		HISEndpoint:    values["his_endpoint"],
		RecareEndpoint: values["recare_endpoint"],
		HISAPIKey:      values["his_api_key"],
		RecareAPIKey:   values["recare_api_key"],
	}

	// easier to debug config problems if we can see what got loaded
	log.Printf("loaded config: %+v", c)

	return c
}

// ---- HIS-side data model (legacy export format) ------------------------

type HISRecord struct {
	PatientID     string  `json:"PatientID"`
	PatientName   string  `json:"PatientName"`
	DOB           string  `json:"DOB"`
	InsuranceNr   string  `json:"InsuranceNr"`
	DischargeDate string  `json:"DischargeDate"`
	Diagnosis     string  `json:"Diagnosis"` // comma-separated ICD-10 codes, or empty
	WeightKg      float64 `json:"WeightKg"`  // 0 if not recorded in kg
	WeightLbs     float64 `json:"WeightLbs"` // 0 if not recorded in lbs
	Notes         string  `json:"Notes"`
}

// ---- Recare-side data model (target shape) ------------------------------

type DischargeSummary struct {
	PatientID     string   `json:"patientId"`
	PatientName   string   `json:"patientName"`
	DOB           string   `json:"dob"`
	InsuranceNr   string   `json:"insuranceNr"`
	DischargeDate string   `json:"dischargeDate"`
	Conditions    []string `json:"conditions"`
	WeightKg      float64  `json:"weightKg"`
	Notes         string   `json:"notes"`
}

// ---- HIS client ----------------------------------------------------------

func fetchRecords() []HISRecord {
	req, _ := http.NewRequest("GET", cfg.HISEndpoint+"/records", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.HISAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("failed to fetch records from HIS: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var records []HISRecord
	if err := json.Unmarshal(body, &records); err != nil {
		log.Fatalf("failed to parse HIS response: %v", err)
	}
	return records
}

// ---- transform -----------------------------------------------------------

// transform maps a raw HIS record into the shape Recare expects.
func transform(rec HISRecord) DischargeSummary {
	return DischargeSummary{
		PatientID:     rec.PatientID,
		PatientName:   rec.PatientName,
		DOB:           rec.DOB,
		InsuranceNr:   rec.InsuranceNr,
		DischargeDate: rec.DischargeDate,
		Conditions:    []string{rec.Diagnosis},
		WeightKg:      rec.WeightKg,
		Notes:         rec.Notes,
	}
}

// ---- Recare client ---------------------------------------------------------

func sendToRecare(summary DischargeSummary) error {
	payload, _ := json.Marshal(summary)

	req, _ := http.NewRequest("POST", cfg.RecareEndpoint+"/discharge-summaries", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.RecareAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("recare returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// logFailedRecord keeps a local copy of anything that failed to send so we
// can retry or inspect it later.
func logFailedRecord(rec HISRecord) {
	f, err := os.OpenFile("failed_records.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("could not open failed_records.log: %v", err)
		return
	}
	defer f.Close()

	raw, _ := json.Marshal(rec)
	f.Write(raw)
	f.Write([]byte("\n"))
}

// ---- main ------------------------------------------------------------------

var recordsProcessed int
var recordsFailed int

func main() {
	cfg = loadConfig("config.yaml")

	records := fetchRecords()
	log.Printf("fetched %d records from HIS", len(records))

	for _, rec := range records {
		// handy for debugging mapping issues in the field
		raw, _ := json.Marshal(rec)
		log.Printf("processing record: %s", string(raw))

		summary := transform(rec)

		if err := sendToRecare(summary); err != nil {
			log.Printf("failed to send record %s: %v", rec.PatientID, err)
			logFailedRecord(rec)
			recordsFailed++
			continue
		}

		recordsProcessed++
	}

	log.Printf("done: %d sent, %d failed", recordsProcessed, recordsFailed)
}

// lbsToKg is here but unused for now — nobody got around to wiring it up
// for the records that only have WeightLbs.
func lbsToKg(lbs float64) float64 {
	kg, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", lbs*0.453592), 64)
	return kg
}
