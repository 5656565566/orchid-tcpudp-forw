package api

import "net/http"

func RegisterRoutes() {
	http.HandleFunc("/api/add", ApiAddMapping)
	http.HandleFunc("/api/delete", ApiDeleteMapping)
	http.HandleFunc("/api/query", ApiQueryMappings)
}
