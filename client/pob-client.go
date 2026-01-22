package client

import (
	"bpl/config"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func GetPoBExport(characterData *Character) (*PathOfBuilding, string, error) {
	jsonData, err := json.Marshal(characterData)
	if err != nil {
		return nil, "", err
	}
	request, err := http.NewRequest("POST", config.Env().POBServerURL, bytes.NewReader(jsonData))
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

func UpdatePoBExport(pobString string) (string, error) {
	// send text to pob server and get updated export
	request, err := http.NewRequest("POST", config.Env().POBServerURL+"/update-config", bytes.NewReader([]byte(pobString)))
	if err != nil {
		return "", err
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	updatedExport := string(body)
	return updatedExport, nil
}
