package handlers

import (
	"context"
	"encoding/json"
	"images-resizing-service/config"
	"images-resizing-service/models"
	"images-resizing-service/services"
	"io"
	"log"
	"net/http"
	"time"
)

func ResizeHandler(s *services.ResizingService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Expecting POST request"))
			return
		}

		request := models.ResizeRequest{}
		err := json.NewDecoder(io.LimitReader(r.Body, 8*1024)).Decode(&request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Failed to parse request"))
			return
		}

		query := r.URL.Query()
		async := query.Get("async")

		var results []models.ResizeResult
		if async == "true" {
			results = s.SubmitForAsyncProcessing(request)
		} else { //default is async is false or null
			results, err = s.ProcessResizes(request)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Failed to process request"))
				return
			}
		}

		data, err := json.Marshal(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to marshal response"))
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Add("content-type", "application/json")
		w.Write(data)
	})
}

func GetImageHandler(s *services.ResizingService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set a timeout for the request context
		ctx, cancel := context.WithTimeout(r.Context(), config.DefaultTimeout)
		defer cancel()
		log.Print("Fetching ", r.URL.String())
		for {
			select {
			case <-ctx.Done():
				// Context has timed out
				w.WriteHeader(http.StatusRequestTimeout)
				w.Write([]byte("Request timeout"))
				return
			default:
				_, isImageInProgress := s.ImagesInProgress.Load(r.URL.String())
				if isImageInProgress {
					time.Sleep(1 * time.Second) //Resizing is still in progress will retry in 1 second...
				} else {
					// Attempt to get the data from the cache
					data, ok := s.Cache.Get(r.URL.String())
					if ok {
						// If data is found, write the response and exit
						w.Header().Set("Content-Type", "image/jpeg")
						w.WriteHeader(http.StatusOK)
						w.Write(data.([]byte))
						return
					} else {
						w.WriteHeader(http.StatusBadRequest)
						w.Write([]byte("No Image with such id is available"))
						return
					}
				}
			}
		}
	})
}
