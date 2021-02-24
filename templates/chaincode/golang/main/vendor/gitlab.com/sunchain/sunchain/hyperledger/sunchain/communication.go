package sunchain

import (
	"net/http"
	"fmt"
	"encoding/json"
)

type MeterPack struct {
	Meters     []string `json:"meters"`
	PackNumber int      `json:"pack_number"`
}

type MeterPacks []MeterPack

const (
	API_USERNAME = "database_handler"
	API_PASSWD   = "7311279a111b2675f93a042a707365b8"
)

func GetPackedOperations() (map[string]struct{}, error) {
	var tempOperations []string
	operationsToReturn := make(map[string]struct{})

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://datahandler.sunchain.fr/operations/pack", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(API_USERNAME, API_PASSWD)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		fmt.Errorf("[ERROR] status code is not 200 : %s", res.Status)
		return nil, nil
	}
	err = json.NewDecoder(res.Body).Decode(&tempOperations)
	if err != nil {
		return nil, err
	}
	for _, ope := range tempOperations {
		operationsToReturn[ope] = struct{}{}
	}

	return operationsToReturn, nil
}

func GetMeterInPack(operationID string) (map[int][]string, error) {
	ordererMeterPack := make(map[int][]string)
	var meterPacksToReturn MeterPacks

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://datahandler.sunchain.fr/operation/" + operationID + "/pack/meters/anonymized", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(API_USERNAME, API_PASSWD)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		fmt.Errorf("[ERROR] status code is not 200 : %s", res.Status)
		return nil, nil
	}

	err = json.NewDecoder(res.Body).Decode(&meterPacksToReturn)
	if err != nil {
		return nil, err
	}

	// DEBUG
	//fakeResp := []byte("[{\"meters\":[\"11111111111111_11111\", \"22222222222222_22222\", \"44444444444444_44444\"],\"pack_number\":1},{\"meters\":[\"33333333333333_33333\"],\"pack_number\":2}]")
	//json.Unmarshal(fakeResp, &meterPacksToReturn)
	for _, value := range meterPacksToReturn {
		ordererMeterPack[value.PackNumber] = value.Meters
	}

	return ordererMeterPack, nil
}
