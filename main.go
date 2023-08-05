package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type Resource struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type BusinessHour struct {
	Id         string `json:"id"`
	ResourceId string `json:"resource_id"`
	Quantity   int64  `json:"quantity"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

type BlockHour struct {
	Id         string `json:"id"`
	ResourceId string `json:"resource_id"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

type Duration struct {
	Seconds int64 `json:"seconds"`
}

type ListBusinessHoursRequest struct {
	ResourceId string `json:"resourceId"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type Appointment struct {
	Id         string `json:"id"`
	ResourceId string `json:"resource_id"`
	Quantity   int64  `json:"quantity"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

type ListBlockHoursRequest struct {
	ResourceId string `json:"resourceId"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type ListAppointmentRequest struct {
	ResourceId string `json:"resourceId"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

// https://github.com/Mohit-Nathrani-at-appointy/test-api/blob/main/structs.go
func StringToTime(timeStr string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, err
	}
	fmt.Println(t)
	return t, nil
}

func TimeToString(tm time.Time) string {
	return tm.Format(time.RFC3339)
}

func fetchDataFromAPI(url string, queryParams map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func GetBlockHours(resourceId, startTime, endTime string) ([]BlockHour, error) {
	apiURL := "http://api.internship.appointy.com:8000/v1/block-hours"
	queryParams := map[string]string{
		"resourceId": resourceId,
		"startTime":  startTime,
		"endTime":    endTime,
	}

	data, err := fetchDataFromAPI(apiURL, queryParams)
	if err != nil {
		return nil, err
	}

	// Unmarshal
	var blockHours []BlockHour
	json.Unmarshal(data, &blockHours)
	if err != nil {
		return nil, err
	}

	return blockHours, nil
}

func GetBusinessHours(resourceId, startTime, endTime string) ([]BusinessHour, error) {

	apiURL := "http://api.internship.appointy.com:8000/v1/business-hours"
	queryParams := map[string]string{
		"resourceId": resourceId,
		"startTime":  startTime,
		"endTime":    endTime,
	}

	data, err := fetchDataFromAPI(apiURL, queryParams)
	if err != nil {
		return nil, err
	}

	var businessHours []BusinessHour
	json.Unmarshal(data, &businessHours)
	if err != nil {
		return nil, err
	}

	return businessHours, nil
}

func GetAppointments(resourceId, startTime, endTime string) ([]Appointment, error) {
	apiURL := "http://api.internship.appointy.com:8000/v1/appointments"
	queryParams := map[string]string{
		"resourceId": resourceId,
		"startTime":  startTime,
		"endTime":    endTime,
	}

	data, err := fetchDataFromAPI(apiURL, queryParams)
	if err != nil {
		return nil, err
	}

	// Unmarshal
	var appointments []Appointment
	json.Unmarshal(data, &appointments)
	if err != nil {
		return nil, err
	}

	return appointments, nil
}

// function to find available slots
func FindAvailableSlots(businessHours []BusinessHour, blockHours []BlockHour, appointments []Appointment, date time.Time, duration time.Duration, quantity int64) []struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
} {
	var availableSlots []struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	overlap := func(start1, end1, start2, end2 time.Time) bool {
		return start1.Before(end2) && end1.After(start2)
	}

	for start := startOfDay; start.Before(endOfDay); start = start.Add(duration) {
		end := start.Add(duration)
		isAvailable := true
		for _, bh := range businessHours {
			bhStart, _ := StringToTime(bh.StartTime)
			bhEnd, _ := StringToTime(bh.EndTime)

			if start.Hour() >= bhStart.Hour() && end.Hour() <= bhEnd.Hour() {
				for _, block := range blockHours {
					blockStart, _ := StringToTime(block.StartTime)
					blockEnd, _ := StringToTime(block.EndTime)
					if overlap(start, end, blockStart, blockEnd) {
						isAvailable = false
						break
					}
				}

				if isAvailable {
					for _, app := range appointments {
						appStart, _ := StringToTime(app.StartTime)
						appEnd, _ := StringToTime(app.EndTime)
						if overlap(start, end, appStart, appEnd) {
							isAvailable = false
							break
						}
					}
				}

				if isAvailable && bh.Quantity >= quantity {
					availableSlots = append(availableSlots, struct {
						StartTime string `json:"start_time"`
						EndTime   string `json:"end_time"`
					}{
						StartTime: TimeToString(start),
						EndTime:   TimeToString(end),
					})
				}
			}
		}
	}
	return availableSlots
}

func FindAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	resourceId := r.URL.Query().Get("resourceId")
	dateStr := r.URL.Query().Get("date")
	durationStr := r.URL.Query().Get("duration")
	quantityStr := r.URL.Query().Get("quantity")

	duration, err := time.ParseDuration(durationStr + "m")
	if err != nil {
		http.Error(w, "Invalid duration", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.ParseInt(quantityStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid quantity", http.StatusBadRequest)
		return
	}

	businessHours, err := GetBusinessHours(resourceId, date.Format("15:04:05"), date.Add(24*time.Hour).Format("15:04:05"))
	if err != nil {
		http.Error(w, "Error fetching business hours", http.StatusInternalServerError)
		return
	}

	blockHours, err := GetBlockHours(resourceId, date.Format("15:04:05"), date.Add(24*time.Hour).Format("15:04:05"))
	if err != nil {
		http.Error(w, "Error fetching block hours", http.StatusInternalServerError)
		return
	}

	appointments, err := GetAppointments(resourceId, date.Format("15:04:05"), date.Add(24*time.Hour).Format("15:04:05"))
	if err != nil {
		http.Error(w, "Error fetching appointments", http.StatusInternalServerError)
		return
	}

	availableSlots := FindAvailableSlots(businessHours, blockHours, appointments, date, duration, quantity)

	response := struct {
		AvailableSlots []struct {
			StartTime string `json:"start_time"`
			EndTime   string `json:"end_time"`
		} `json:"available_slots"`
		TimeTaken string `json:"time_taken"`
	}{

		AvailableSlots: availableSlots,
		TimeTaken:      time.Now().UTC().Format(time.RFC3339),
	}

	// Write the JSON response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/v1/availability", FindAvailabilityHandler)

	fmt.Println("Server listening on port 8000...")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
