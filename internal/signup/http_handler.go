package signup

import (
	"encoding/json"
	"errors"
	"net/http"
)

func NewHandler(service *VerificationService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /signup", func(w http.ResponseWriter, r *http.Request) {
		var request SignupRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := service.Start(r.Context(), request)
		if err != nil {
			status := http.StatusBadGateway
			var upstream *InfraiError
			if errors.As(err, &upstream) && upstream.Status >= 400 && upstream.Status < 500 {
				status = http.StatusUnprocessableEntity
			}
			if err.Error() == "email, password, and name are required" {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	return mux
}
