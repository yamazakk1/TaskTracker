package handlers

import "net/http"
import "encoding/json"



type Payload struct{
	Key string 
	Payload any
}

func toPayload(key string, pl any) Payload{
	return Payload{Key: key, Payload: pl}
}

func toJSON(storage map[string]any, payload Payload){
	storage[payload.Key] = payload.Payload
}

func responseWithJSON(w http.ResponseWriter, code int, payload ...Payload){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	storage := make(map[string]any)
	for _, pl:= range payload{
		toJSON(storage, pl)
	}
	json.NewEncoder(w).Encode(storage)
}

func responseWithError(w http.ResponseWriter, code int, message string){
	responseWithJSON(w,code, toPayload("error", message))
}



