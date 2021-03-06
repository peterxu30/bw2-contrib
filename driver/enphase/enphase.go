package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/parnurzeal/gorequest"
)

const uriBase = "https://api.enphaseenergy.com/api/v2"

type Enphase struct {
	APIKey string
	UserID string
	req    *gorequest.SuperAgent
	sysNum uint64
}

type enphaseSystem struct {
	SystemID         uint64 `json:"system_id"`
	SystemName       string `json:"system_name"`
	SystemPublicName string `json:"system_public_name"`
	Status           string `json:"status"`
	Timezone         string `json:"timezone"`
	Country          string `json:"country"`
	State            string `json:"state"`
	City             string `json:"city"`
}

type indexResult struct {
	Systems []enphaseSystem `json:"systems"`
}

type Summary struct {
	CurrentPower   uint64 `json:"current_power"`
	EnergyLifetime uint64 `json:"energy_lifetime"`
	EnergyToday    uint64 `json:"energy_today"`
	Modules        uint64 `json:"modules"`
	OperationalAt  uint64 `json:"operational_at"`
	SizeWatts      uint64 `json:"size_w"`
	Source         string `json:"source"`
	Status         string `json:"status"`
	SummaryDate    string `json:"summary_date"`
	SystemID       uint64 `json:"system_id"`
}

func NewEnphase(apiKey string, userID string, sysName string) (*Enphase, error) {
	request := gorequest.New()
	resp, body, errs := request.Get(uriBase + "/systems").
		Query("key=" + apiKey).
		Query("user_id=" + userID).
		EndBytes()
	if errs != nil {
		return nil, fmt.Errorf("System index request failed: %v", errs)
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("System index request failed: %v", resp.Status)
	}

	defer resp.Body.Close()
	var rslt indexResult
	err := json.Unmarshal(body, &rslt)
	if err != nil {
		return nil, fmt.Errorf("System index request failed: %v", err)
	}
	for _, system := range rslt.Systems {
		if system.SystemName == sysName {
			return &Enphase{
				APIKey: apiKey,
				UserID: userID,
				req:    request,
				sysNum: system.SystemID,
			}, nil
		}
	}

	return nil, errors.New("No system with specified name found")
}

func (enphase *Enphase) ReadSummary() (*Summary, error) {
	resp, body, errs := enphase.req.Get(fmt.Sprintf("%s/systems/%d/summary", uriBase, enphase.sysNum)).
		Query("key=" + enphase.APIKey).
		Query("user_id=" + enphase.UserID).
		EndBytes()
	if errs != nil {
		return nil, fmt.Errorf("Summary request failed: %v", errs)
	}

	defer resp.Body.Close()
	var summ Summary
	err := json.Unmarshal(body, &summ)
	if err != nil {
		return nil, errors.New("Received invalid summary as response")
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Summary request failed: %v", err)
	}

	return &summ, nil
}

func (enphase *Enphase) PollSummary(dur time.Duration) chan *Summary {
	summCh := make(chan *Summary)
	go func() {
		for {
			summary, err := enphase.ReadSummary()
			if err != nil {
				fmt.Printf("Error polling enphase data: %v\n", err)
				close(summCh)
				return
			}
			summCh <- summary
			time.Sleep(dur)
		}
	}()

	return summCh
}
