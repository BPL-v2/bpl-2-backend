package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

func GetPoBExport(characterData *Character) (*PathOfBuilding, string, error) {
	jsonData, err := json.Marshal(characterData)
	if err != nil {
		return nil, "", err
	}
	request, err := http.NewRequest("POST", os.Getenv("POB_SERVER_URL"), bytes.NewReader(jsonData))
	if err != nil {
		return nil, "", err
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}
	export := string(body)
	pob, err := DecodePoBExport(export)
	if err != nil {
		return nil, "", err
	}
	return pob, export, nil
}
