package mc

import (
	"log"
	"net/http"
	"nexus/data/mc"
	"strconv"
)

func (a *ReconApp) handleStatus(response http.ResponseWriter, request *http.Request) {
	key, err := mc.GetEntityKey(request.Context(), request.FormValue("key"), a.DB)
	if err != nil {
		log.Printf("mc.GetEntityKey(%q) err: %v", request.FormValue("key"), err)
		response.WriteHeader(500)
		return
	}

	status := request.FormValue("status")
	bat, err := strconv.Atoi(request.FormValue("bat"))
	if err != nil {
		log.Printf("strconv err: %v", err)
		response.WriteHeader(403)
		return
	}

	log.Printf("Got status update for %s(batt=%d): %s", key.Name, bat, status)
	s := &mc.Status{
		EntityKeyUID: key.UID,
		BatteryLevel: bat,
		Status:       status,
		IsHeartbeat:  false,
	}
	_, err = mc.CreateStatus(request.Context(), s, a.DB)
	if err != nil {
		log.Printf("mc.CreateStatus(%+v) err: %v", s, err)
		response.WriteHeader(500)
		return
	}
}

func (a *ReconApp) handleHeartbeat(response http.ResponseWriter, request *http.Request) {
	key, err := mc.GetEntityKey(request.Context(), request.FormValue("key"), a.DB)
	if err != nil {
		log.Printf("mc.GetEntityKey(%q) err: %v", request.FormValue("key"), err)
		response.WriteHeader(500)
		return
	}

	bat, err := strconv.Atoi(request.FormValue("bat"))
	if err != nil {
		log.Printf("strconv err: %v", err)
		response.WriteHeader(403)
		return
	}

	log.Printf("Got heartbeat for %s(batt=%d)", key.Name, bat)
	s := &mc.Status{
		EntityKeyUID: key.UID,
		BatteryLevel: bat,
		IsHeartbeat:  true,
	}
	_, err = mc.CreateStatus(request.Context(), s, a.DB)
	if err != nil {
		log.Printf("mc.CreateStatus(%+v) err: %v", s, err)
		response.WriteHeader(500)
		return
	}
}

func (a *ReconApp) handleLocationUpdate(response http.ResponseWriter, request *http.Request) {
	key, err := mc.GetEntityKey(request.Context(), request.FormValue("key"), a.DB)
	if err != nil {
		log.Printf("mc.GetEntityKey(%q) err: %v", request.FormValue("key"), err)
		response.WriteHeader(500)
		return
	}

	lat, err := strconv.ParseFloat(request.FormValue("lat"), 64)
	if err != nil {
		log.Printf("strconv err for field 'lat': %v", err)
		response.WriteHeader(403)
		return
	}
	lon, err := strconv.ParseFloat(request.FormValue("lon"), 64)
	if err != nil {
		log.Printf("strconv err for field 'lon': %v", err)
		response.WriteHeader(403)
		return
	}
	kph, err := strconv.ParseFloat(request.FormValue("kph"), 64)
	if err != nil {
		log.Printf("strconv err for field 'kph': %v", err)
		response.WriteHeader(403)
		return
	}
	course, err := strconv.Atoi(request.FormValue("course"))
	if err != nil {
		log.Printf("strconv err for field 'course': %v", err)
		response.WriteHeader(403)
		return
	}
	acc, err := strconv.Atoi(request.FormValue("acc"))
	if err != nil {
		log.Printf("strconv err for field 'acc': %v", err)
		response.WriteHeader(403)
		return
	}

	log.Printf("Got location for %s: lat=%.6f lon=%.6f kph=%.2f heading=%d accuracy=%d", key.Name, lat, lon, kph, course, acc)
	loc := &mc.Location{
		EntityKeyUID: key.UID,
		Lat:          lat,
		Lon:          lon,
		Kph:          kph,
		Course:       course,
		Accuracy:     acc,
	}
	_, err = mc.CreateLocation(request.Context(), loc, a.DB)
	if err != nil {
		log.Printf("mc.CreateLocation(%+v) err: %v", loc, err)
		response.WriteHeader(500)
		return
	}
}
