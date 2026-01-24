package handlers

import "net/http"
import "encoding/json"

func responseWithJSON(w http.ResponseWriter, code int, payload any){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func responseWithError(w http.ResponseWriter, code int, message string){
	responseWithJSON(w,code, map[string]string{"error":message})
}

func healthCheck(w http.ResponseWriter){
	responseWithJSON(w, http.StatusOK,map[string]string{"status": "ok"})
}

