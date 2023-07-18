package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type AccessLog struct {
	Time        string `json:"time"`
	IP          string `json:"ip"`
	Path        string `json:"path"`
	UserAgent   string `json:"user_agent"`
	GeoLocation string `json:"geo_location"`
}

func logRequestData(r *http.Request) {
	requestTime := time.Now().Format(time.ANSIC)

	// Get the user's IP address from the CF-Connecting-IP header
	ipAddress := r.Header.Get("CF-Connecting-IP")
	if ipAddress == "" {
		// If CF-Connecting-IP header is not available, fallback to RemoteAddr
		ipAddress, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	userAgent := r.UserAgent()

	requestedPath := strings.TrimPrefix(r.URL.Path, "/")

	geoLocation := getGeoLocation(ipAddress)

	logData := AccessLog{
		Time:        requestTime,
		IP:          ipAddress,
		Path:        requestedPath,
		UserAgent:   userAgent,
		GeoLocation: geoLocation,
	}

	fmt.Printf("[%s] IP: %s Requested: %s, GeoLocation: %s\n",
		requestTime, ipAddress, requestedPath, geoLocation)

	writeAccessLog(logData)
}

func getGeoLocation(ipAddress string) string {
	response, err := http.Get(ipAPIEndpoint + ipAddress)
	if err != nil {
		log.Printf("Error fetching geo-location: %s", err)
		return "Unknown"
	}
	defer response.Body.Close()

	var geoInfo struct {
		Success string `json:"status"`
		City    string `json:"city"`
		Region  string `json:"regionName"`
		Country string `json:"country"`
	}

	err = json.NewDecoder(response.Body).Decode(&geoInfo)
	if err != nil {
		log.Printf("Error parsing geo-location data: %s", err)
		return "Unknown"
	}

	if strings.Compare(geoInfo.Success, "success") != 0 {
		return "UNABLE TO GET"
	}

	geoLocation := fmt.Sprintf("%s, %s, %s", geoInfo.City, geoInfo.Region, geoInfo.Country)
	return geoLocation
}

func writeAccessLog(data AccessLog) {
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Error opening log file: %s", err)
		return
	}
	defer logFile.Close()

	logData, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Printf("Error marshaling access log data: %s", err)
		return
	}

	logFile.Write(logData)
	logFile.Write([]byte("\n"))
}
