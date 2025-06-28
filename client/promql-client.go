package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

type PromQLResponse struct {
	Status    string  `json:"status"`
	ErrorType *string `json:"errorType"`
	Error     *string
	Data      *struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				CharacterName string `json:"CharacterName"`
				Name          string `json:"__name__"`
				Instance      string `json:"instance"`
				Job           string `json:"job"`
			} `json:"metric"`
			Values []any `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

type RespValue struct {
	TimeStamp    float64
	Armour       float64
	Evasion      float64
	EnergyShield float64
	HP           float64
	Mana         float64
	PhysMaxHit   float64
	EleMaxHit    float64
}

type StatValues map[string][]any

func fetchData(metric string, characterName string, diffInSeconds int, end time.Time) (PromQLResponse, error) {
	query := fmt.Sprintf("%s{CharacterName=\"%s\"}[%ds:1m]", metric, characterName, diffInSeconds)
	url := fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s&time=%d", query, end.Unix())
	fmt.Println(url)
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	var promQLResponse PromQLResponse
	err = json.Unmarshal(body, &promQLResponse)
	if err != nil {
		log.Fatal(err)
	}
	return promQLResponse, nil
}

func GetCharacterMetrics(characterName string, metrics []string, from time.Time, to time.Time) StatValues {
	diffInSeconds := int(math.Round(to.Sub(from).Seconds()))
	statValues := StatValues{}

	responseChan := make(chan PromQLResponse, len(metrics))
	wg := sync.WaitGroup{}
	for _, metric := range metrics {
		wg.Add(1)
		go func(metric string) {
			defer wg.Done()
			promQLResponse, err := fetchData(metric, characterName, diffInSeconds, to)
			if err != nil {
				log.Printf("Error fetching data: %s", err)
				return
			}
			responseChan <- promQLResponse
		}(metric)
	}
	wg.Wait()
	close(responseChan)
	for promQLResponse := range responseChan {
		fmt.Println(promQLResponse)

		if promQLResponse.Error != nil {
			log.Printf("Error fetching data: %s", *promQLResponse.Error)
			continue
		}
		if promQLResponse.Data == nil || len(promQLResponse.Data.Result) == 0 {
			continue
		}
		result := promQLResponse.Data.Result[0]
		statValues[result.Metric.Name] = result.Values
	}
	return statValues
}
