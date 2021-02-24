package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"gitlab.com/sunchain/sunchain/hyperledger/sunchain"
)

//getMetersFlow gets all the meters of the blockchain and marshals the response
func getMetersFlow(stub shim.ChaincodeStubInterface) pb.Response {

	meters, err := getMeters(stub)
	if err != nil {
		return logReturn(err, "")
	}

	buf, err := json.Marshal(meters)
	if err != nil {
		return logReturn(err, "Couldn't marshal the meters")
	}

	return logSuccess(buf, "GetMeters returned %v meters.", len(meters))
}

func getMeters(stub shim.ChaincodeStubInterface) (map[string]sunchain.Meter, error) {
	meters := make(map[string]sunchain.Meter)
	var mtr sunchain.Meter

	metersIterator, err := stub.GetStateByPartialCompositeKey(meterKey, []string{})
	if err != nil {
		return nil, fmt.Errorf("couldn't access state for meters : %s", err)
	}
	defer metersIterator.Close()

	for metersIterator.HasNext() {
		kv, err := metersIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("couldn't get next meters from iterator : %s", err)
		}
		err = json.Unmarshal(kv.Value, &mtr)
		if err != nil {
			return nil, fmt.Errorf("couldn't unmarshal payload from meters state: %s", err)
		}
		cP := runeCP(mtr.ConsoProd)
		meters[mtr.ID+string(cP)] = mtr
	}

	return meters, nil
}

//getMeasureFlow extract the measure from the blockchain and marshals it
func getMeasureFlow(stub shim.ChaincodeStubInterface, meterID string, timestamp time.Time, consoProd string) pb.Response {

	measure, err := getMeasure(stub, meterID, timestamp, consoProd)
	if err != nil {
		return logReturn(err, "")
	}

	buf, err := json.Marshal(measure)
	if err != nil {
		return logReturn(err, "Couldn't marshal response for measure")
	}

	return logSuccess(buf, "GetMeasure %v", measure)
}

//getMeasure returns the unique measure that a {meterID+consoProd} has at a given time
func getMeasure(stub shim.ChaincodeStubInterface, meterID string, timestamp time.Time, consoProd string) (sunchain.Measure, error) {
	timestamp = timestamp.Truncate(period)
	cP := runeCP(consoProd)
	var m sunchain.Measure

	measureIterator, err := stub.GetStateByPartialCompositeKey(measureIndex, []string{timestamp.Format(time.RFC3339), meterID, string(cP)})
	if err != nil {
		return sunchain.Measure{}, fmt.Errorf("couldn't access state for measure: %s", err)
	}
	defer measureIterator.Close()

	if !measureIterator.HasNext() {
		return sunchain.Measure{}, nil
	}
	for measureIterator.HasNext() {
		kv, err := measureIterator.Next()
		if err != nil {
			return sunchain.Measure{}, fmt.Errorf("couldn't iterate over measures: %s", err)
		}
		err = json.Unmarshal(kv.Value, &m)
		if err != nil {
			return sunchain.Measure{}, fmt.Errorf("couldn't unmarshal measure: %s", err)
		}
	}
	return m, nil
}

func getMeasures(stub shim.ChaincodeStubInterface, timestamp time.Time, meters map[string]sunchain.Meter) (map[string]sunchain.Measure, error) {

	meterID := make([]string, 0)
	for i := range meters {
		meterID = append(meterID, i)
	}
	measureList := make(map[string]sunchain.Measure)
	for i, m := range meters {
		resp, err := getMeasure(stub, m.ID, timestamp, m.ConsoProd)
		if err != nil {
			return nil, err
		}
		measureList[i] = resp
	}

	return measureList, nil
}

//getMeasuresAtFlow returns all the measures that have been made at a given time and marshals the response
func getMeasuresAtFlow(stub shim.ChaincodeStubInterface, timestamp time.Time) pb.Response {
	measureList, err := getMeasuresAndRedistribute(stub, timestamp)
	if err != nil {
		return logReturn(err, "")
	}
	buf, err := json.Marshal(measureList)
	if err != nil {
		return logReturn(err, "Couldn't marshal measures")
	}
	return logSuccess(buf, "GetMeasuresAt %v returned the result of %v meters.", timestamp, len(measureList))
}

//getMeasuresAndRedistribute gets all the measures for all meters and calculates redistribution
func getMeasuresAndRedistribute(stub shim.ChaincodeStubInterface, timestamp time.Time) (map[string]sunchain.Measure, error) {
	meters, err := getMeters(stub)
	if err != nil {
		return nil, err
	}
	measures, err := getMeasures(stub, timestamp, meters)
	if err != nil {
		return nil, err
	}

	operationPerPack, err := sunchain.GetPackedOperations()
	if err != nil {
		return nil, err
	}

	//no operator on array to insert data only if it's unique
	operations := make(map[string]struct{})
	for _, v := range meters {
		operations[v.OperationID] = struct{}{}
	}

	temp := make(map[string]sunchain.Measure)
	for ope := range operations {
		// check operation name to get the right division calculation
		if _, ok := operationPerPack[ope]; ok {
			temp, err = packedRedistributionCalculation(meters, measures, ope)
			if err != nil {
				return nil, err
			}
		} else {
			temp = redistributionCalculation(meters, measures, ope)
		}
		for i, v := range temp {
			measures[i] = v
		}
	}

	return measures, nil
}

// getMeasuresBetweenFlow gets all measures in the interval given and marshals the response
func getMeasuresBetweenFlow(stub shim.ChaincodeStubInterface, start, end time.Time) pb.Response {

	if start.After(end) || start.Equal(end) {
		return logReturn(fmt.Errorf("start >= end : %v %v", start, end), "")
	}
	measures, err := getMeasuresBetween(stub, start, end)
	if err != nil {
		return logReturn(err, "")
	}
	buf, err := json.Marshal(measures)
	if err != nil {
		return logReturn(err, "GetMeasuresBetween couldn't marshal data to JSON")
	}
	return logSuccess(buf, "GetMeasuresBetween %s and %s returned results for %d meters", start, end, len(measures))
}

func getMeasuresBetween(stub shim.ChaincodeStubInterface, start, end time.Time) (map[string][]sunchain.Measure, error) {
	ms := make(map[string][]sunchain.Measure)

	for _, t := range discretePeriod(start, end) {
		m, err := getMeasuresAndRedistribute(stub, t)
		if err != nil {
			return nil, err
		}
		zm := sunchain.Measure{}
		for k, v := range m {
			if v == zm {
				continue
			}
			ms[k] = append(ms[k], v)

		}
	}
	return ms, nil
}
