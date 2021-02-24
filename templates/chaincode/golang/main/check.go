package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"gitlab.com/sunchain/sunchain/hyperledger/sunchain"
)

// checkMeter checks the inputs if adding a new meter
func checkMeter(operationID, meterID, consoProd string) error {
	defer timeTrack(time.Now(), "checkMeter")
	if len(operationID) != 11 {
		return fmt.Errorf("operationID hasn't the required size (11 characters) : %s", operationID)
	}
	if len(meterID) != 20 {
		return fmt.Errorf("meterID hasn't the required size (20 digits GUID) : %s", meterID)
	}
	if consoProd != "Conso" && consoProd != "Prod" {
		return fmt.Errorf("consoProd isn't equal to Conso or Prod : %s", consoProd)
	}
	return nil
}

// checkLast returns the last measure for this {meterID+indexName+consoProd} tuple, or an empty one
func checkLast(stub shim.ChaincodeStubInterface, meterID, indexName, consoProd string) (sunchain.Measure, error) {
	defer timeTrack(time.Now(), "checkLast")
	var err error
	var lastDate []byte
	var lastMeasureDate time.Time
	var m sunchain.Measure
	cP := runeCP(consoProd)

	lastDate, err = stub.GetState("LAST_" + meterID + string(cP) + indexName)
	if err != nil {
		return sunchain.Measure{}, fmt.Errorf("couldn't access LAST state: %s", err)
	}
	if lastDate == nil {
		return sunchain.Measure{}, nil
	}

	lastMeasureDate, err = time.Parse(time.RFC3339, string(lastDate))
	if err != nil {
		return sunchain.Measure{}, fmt.Errorf("couldn't parse lastDate into time: %s", err)
	}
	m, err = getMeasure(stub, meterID, lastMeasureDate, consoProd)
	if err != nil {
		return sunchain.Measure{}, fmt.Errorf("couldn't get measure at LAST: %s", err)
	}
	return m, nil
}

// checkFirst checks if it is the first entry for that {meterID+indexName+consoProd} tuple
func checkFirst(stub shim.ChaincodeStubInterface, meterID, indexName, consoProd string) (time.Time, error) {
	defer timeTrack(time.Now(), "checkFirst")
	var err error
	var firstDate []byte
	var firstTime time.Time

	firstDate, err = stub.GetState("FIRST_" + meterID + indexName + consoProd)
	if err != nil {
		return time.Time{}, fmt.Errorf("couldn't access FIRST: %s", err)
	}

	if firstDate == nil {
		return time.Time{}, nil
	}
	firstTime, err = time.Parse(time.RFC3339, string(firstDate))
	if err != nil {
		return time.Time{}, fmt.Errorf("couldn't parse FIRST: %s", err)
	}

	return firstTime, nil
}

// checkIndexChange scans since the last time the tuple {meterID+indexName+consoProd} was written that the couple {meterID+consoProd}
// has been registered. If yes, that means that that couple entered data with another index.
func checkIndexChange(stub shim.ChaincodeStubInterface, meterID, consoProd string, start time.Time) (bool, error) {
	defer timeTrack(time.Now(), "checkIndexChange")
	var lastMeterTimestamp time.Time
	var lmt string
	cP := runeCP(consoProd)

	lastDateIterator, err := stub.GetStateByPartialCompositeKey("LAST_", []string{meterID + string(cP)})
	if err != nil {
		return false, err
	}
	defer lastDateIterator.Close()
	for lastDateIterator.HasNext() {
		kv, err := lastDateIterator.Next()
		if err != nil {
			return false, err
		}
		err = json.Unmarshal(kv.Value, &lmt)
		lastMeterTimestamp, err = time.Parse(time.RFC3339, lmt)
	}

	if lastMeterTimestamp.After(start) {
		return true, nil
	}
	return false, nil
}
