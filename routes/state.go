package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/victorjacobs/go-comfoair/bridge"
	"github.com/victorjacobs/go-comfoair/comfoair"
)

type stateResponse struct {
	FilterDays    float32   `json:"filter_days"`
	LowDays       float32   `json:"low_days"`
	MediumDays    float32   `json:"medium_days"`
	HighDays      float32   `json:"high_days"`
	TotalDays     float32   `json:"total_days"`
	LastRefreshed time.Time `json:"last_refreshed"`
}

type cache struct {
	lastRefreshed int64
	operatingTime *comfoair.OperatingTime
}

func State(b *bridge.Bridge) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	c := &cache{}

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		now := time.Now().UnixMilli()

		if c.lastRefreshed+30_000 < now || c.operatingTime == nil {
			// Refresh
			operatingTime, err := b.GetOperatingTime()
			if err != nil {
				log.Printf("Failed to get operating time: %v", err)

				return
			}

			c.lastRefreshed = now
			c.operatingTime = operatingTime

			log.Printf("Refreshed web cache")
		}

		resp := stateResponse{
			FilterDays:    float32(c.operatingTime.FilterHours) / 24,
			LowDays:       float32(c.operatingTime.LowHours) / 24,
			MediumDays:    float32(c.operatingTime.MediumHours) / 24,
			HighDays:      float32(c.operatingTime.HighHours) / 24,
			TotalDays:     float32(c.operatingTime.LowHours+c.operatingTime.MediumHours+c.operatingTime.HighHours) / 24,
			LastRefreshed: time.Unix(0, c.lastRefreshed*int64(time.Millisecond)),
		}

		if marshaled, err := json.Marshal(resp); err != nil {
			log.Printf("error marshaling: %v", err)
		} else {
			w.Write(marshaled)
		}
	}
}
