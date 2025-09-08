package api

import (
	"encoding/json"
	"net/http"

	"github.com/brinestone/dfs/storage"
)

const storageUrlPath = "storage"

func getStorageStatsHandler(api *Api, store *storage.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, err := json.Marshal(store.GetStats())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			api.Logger.Error("error while handling request", "err", err)
			return
		}
		w.Write(s)
	})
}

func (api *Api) setupStorageEndpointHandlers() {
	api.registerLoggedHandler(api.makeApiPathWithBase("stats", storageUrlPath), getStorageStatsHandler(api, api.Store))
}
