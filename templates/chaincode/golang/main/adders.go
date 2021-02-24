package main

import (
	"encoding/json"
	"fmt"
	"gitlab.com/sunchain/sunchain/hyperledger/sunchain"
	"log"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

// addMeterFlow dictates the flow of adding a new measure -> input check, add to the state, marshals response
func addMeterFlow(stub shim.ChaincodeStubInterface, operationID, meterID, consoProd string) pb.Response {
	defer timeTrack(time.Now(), "addMeterFlow")
	var err error
	var meter sunchain.Meter
	var bytes []byte

	err = checkMeter(operationID, meterID, consoProd)
	if err != nil {
		return logReturn(err, "")
	}

	meter, err = meterInState(stub, operationID, meterID, consoProd)
	if err != nil {
		return logReturn(err, "")
	}

	bytes, err = json.Marshal(meter)
	if err != nil {
		return logReturn(err, "Couldn't marshal meter")
	}
	return logSuccess(bytes, "AddMeter: %v.", meter)
}

// meterInState adds a new meter in the blockchain with the key {meterIndex,meterID,consoProd(C or P)} if not already present.
func meterInState(stub shim.ChaincodeStubInterface, operationID, meterID, consoProd string) (sunchain.Meter, error) {
	defer timeTrack(time.Now(), "meterInState")
	cP := runeCP(consoProd)
	indexKey, err := stub.CreateCompositeKey(meterKey, []string{meterID, string(cP)})
	if err != nil {
		return sunchain.Meter{}, fmt.Errorf("couldn't create composite key for the meter : %s", err)
	}

	//Verifying key unicity
	mtr, err := stub.GetState(indexKey)
	if err != nil {
		return sunchain.Meter{}, fmt.Errorf("couldn't check the state for the meter : %s", err)
	}
	if mtr != nil {
		return sunchain.Meter{}, nil
	}

	// Creating meter
	newMeter := sunchain.Meter{
		ID:          meterID,
		ConsoProd:   consoProd,
		OperationID: operationID,
	}

	// Meter serialization
	newMeterBytes, err := json.Marshal(newMeter)
	if err != nil {
		return sunchain.Meter{}, fmt.Errorf("couldn't serialize the meter : %s", err)
	}

	// Pushing meter in the chaincode
	err = stub.PutState(indexKey, newMeterBytes)
	if err != nil {
		return sunchain.Meter{}, fmt.Errorf("couldn't push the meter in the chaincode : %s", err)
	}
	return newMeter, nil
}

// addMeasureFlow dictates the measure creation -> check inputs, add meter if necessary, add the measure, marshals the response
func addMeasureFlow(stub shim.ChaincodeStubInterface, operationID, meterID, consoProd, indexName string, indexValue int, timestamp time.Time) pb.Response {
	defer timeTrack(time.Now(), "addMeasureFlow")
	var err error
	var measure sunchain.Measure
	var bytes []byte

	if indexValue < 0 {
		return logReturn(fmt.Errorf("negative indexValue : %d", indexValue), "")
	}

	//We add the meter associated to the measure if it doesn't exist
	err = checkMeter(operationID, meterID, consoProd)
	if err != nil {
		return logReturn(err, "")
	}
	_, err = meterInState(stub, operationID, meterID, consoProd)
	if err != nil {
		return logReturn(err, "")
	}

	measure, err = addMeasure(stub, operationID, meterID, consoProd, indexName, indexValue, timestamp)
	if err != nil {
		return logReturn(err, "")
	}

	bytes, err = json.Marshal(measure)
	if err != nil {
		return logReturn(err, "Couldn't marshal measure")
	}

	return logSuccess(bytes, "AddMeasure: %v.", measure)
}

// addSingleMeasure writes the first measure for each meter, and computes the others
func addSingleMeasure(stub shim.ChaincodeStubInterface, operationID, meterID, consoProd, indexName string, indexValue int, timestamp time.Time, precedingMeasure sunchain.Measure) (sunchain.Measure, error) {
	defer timeTrack(time.Now(), "addSingleMeasure")
	measure := sunchain.Measure{
		Meter: sunchain.Meter{
			ID:          meterID,
			ConsoProd:   consoProd,
			OperationID: operationID,
		},
		IndexName:    indexName,
		IndexValue:   indexValue,
		Delta:        0,
		Timestamp:    timestamp,
		Redistribute: 0,
	}
	unset := time.Time{}

	first, err := checkFirst(stub, meterID, indexName, consoProd)
	if err != nil {
		return sunchain.Measure{}, err
	}
	if first == unset || first == timestamp {
		err = stub.PutState("FIRST_"+meterID+indexName+consoProd, []byte(measure.Timestamp.Format(time.RFC3339)))
		if err != nil {
			return sunchain.Measure{}, fmt.Errorf("couldn't put FIRST: %s", err)
		}
		precedingMeasure = sunchain.Measure{}
	}

	delta, err := deltaCalculation(precedingMeasure, measure)
	if err != nil {
		return sunchain.Measure{}, err
	}
	measure.Delta = delta

	err = measureInState(stub, measure)
	if err != nil {
		return sunchain.Measure{}, err
	}
	return measure, nil
}

//measureInState adds the measure in the State
func measureInState(stub shim.ChaincodeStubInterface, measure sunchain.Measure) error {
	defer timeTrack(time.Now(), "measureInState")
	var err error
	fmt.Println(measure)
	cP := runeCP(measure.ConsoProd)
	indexKey, err := stub.CreateCompositeKey(measureIndex, []string{measure.Timestamp.Format(time.RFC3339), measure.ID, string(cP)})
	if err != nil {
		return err
	}

	measureBytes, err := json.Marshal(measure)
	if err != nil {
		return err
	}
	err = stub.PutState(indexKey, measureBytes)
	if err != nil {
		return fmt.Errorf("couldn't save measure to state: %s", err)
	}

	// we update the last measure for that {meterID+indexName+consoProd} tuple
	err = stub.PutState("LAST_"+measure.ID+string(cP)+measure.IndexName, []byte(measure.Timestamp.Format(time.RFC3339)))
	if err != nil {
		return fmt.Errorf("couldn't put to LAST: %s", err)
	}

	//enable better handling of index change
	indexKey, err = stub.CreateCompositeKey("LAST_", []string{measure.ID + string(cP)})
	if err != nil {
		return err
	}

	timestampBytes, err := json.Marshal(measure.Timestamp)
	if err != nil {
		return err
	}
	err = stub.PutState(indexKey, timestampBytes)
	if err != nil {
		return err
	}
	return nil
}

//addMeasure checks the conditions of measure adding. If it is on the same period, the next one or if some data were missing for that meter since several periods.
func addMeasure(stub shim.ChaincodeStubInterface, operationID string, meterID string, consoProd string, indexName string, indexValue int, timestamp time.Time) (sunchain.Measure, error) {
	defer timeTrack(time.Now(), "addMeasure")
	var measure, empty, lastMeasure sunchain.Measure
	var err error
	var index bool
	// TruncateDate truncs the actual date minus the period (10h16-> 10h10 etc.)
	truncatedDate := timestamp.Truncate(period)

	lastMeasure, err = checkLast(stub, meterID, indexName, consoProd)
	if err != nil {
		return empty, err
	}

	//meter's first entry check
	if lastMeasure == empty {
		fmt.Println("CASE INIT")
		measure, err = addSingleMeasure(stub, operationID, meterID, consoProd, indexName, indexValue, truncatedDate, empty)
		if err != nil {
			return empty, err
		}
		return measure, nil
	}

	//prevents old data insertion
	if lastMeasure.Timestamp.After(timestamp) {
		return empty, fmt.Errorf("couldn't add measure before already present data : %v < %v", timestamp, lastMeasure.Timestamp)
	}

	// check same/next period
	if lastMeasure.Timestamp.Add(period) == truncatedDate || lastMeasure.Timestamp == truncatedDate {
		fmt.Println("CASE SAME/NEXT PERIOD")
		measure, err = addSingleMeasure(stub, operationID, meterID, consoProd, indexName, indexValue, truncatedDate, lastMeasure)
		if err != nil {
			return sunchain.Measure{}, errors.Wrap(err, "couldn't add the measure")
		}
		return measure, nil
	}

	//index change check -> delta with new index (e.g : peak times/off-peak times)
	index, err = checkIndexChange(stub, meterID, consoProd, lastMeasure.Timestamp)
	if index {
		fmt.Println("CASE INDEX CHANGE")
		measure, err = addSingleMeasure(stub, operationID, meterID, consoProd, indexName, indexValue, truncatedDate, lastMeasure)
		if err != nil {
			return empty, err
		}
		return measure, nil
	}

	//missing data
	measure, err = interpolation(stub, operationID, meterID, consoProd, indexName, indexValue, lastMeasure, truncatedDate)
	if err != nil {
		return empty, err
	}

	//measure override ensuring last data is right in case of interpolation
	measure, err = addSingleMeasure(stub, operationID, meterID, consoProd, indexName, indexValue, truncatedDate, measure)
	if err != nil {
		return empty, err
	}
	return measure, nil
}

//interpolation applies the method according to the missing data.
// If <1h, linear interpolation, if 1h<data<24h, last week's measures insertion and if >24h, data considered missing.
// In case the meter is newer than a week, we revert to a linear interpolation on all cases.
func interpolation(stub shim.ChaincodeStubInterface, operationID, meterID, consoProd, indexName string, indexValue int, lastMeasure sunchain.Measure, truncatedDate time.Time) (sunchain.Measure, error) {
	defer timeTrack(time.Now(), "interpolation")
	var measure sunchain.Measure

	firstMeasure, err := stub.GetState("FIRST_" + meterID + indexName + consoProd)
	if err != nil {
		return sunchain.Measure{}, fmt.Errorf("could not access FIRST state : %v", err)
	}

	firstMeasureDate, err := time.Parse(time.RFC3339, string(firstMeasure))
	if err != nil {
		return sunchain.Measure{}, fmt.Errorf("couldn't parse FIRST to time.Time format : %v", err)
	}

	// if the hole's duration is less than an hour
	holeHour := lastMeasure.Timestamp.Add(time.Duration(60 * time.Minute)).After(truncatedDate)
	// if the hole is smaller than a day
	holeDay := lastMeasure.Timestamp.Add(time.Duration(1440 * time.Minute)).After(truncatedDate)
	// if the meter is older than a week
	meterWeek := lastMeasure.Timestamp.Add(time.Duration(1440 * 7 * time.Minute)).After(firstMeasureDate)

	log.Println("holeHour:", holeHour, "holeDay:", holeDay, "meterWeek:", meterWeek)

	// hole < 1h, OR hole < day and meter newer than a week
	if holeHour || (!meterWeek && holeDay) {
		fmt.Println("CASE INTERPOLATION1")
		measure, err = linearInterpolation(stub, operationID, meterID, consoProd, indexName, indexValue, lastMeasure, truncatedDate)
		if err != nil {
			return sunchain.Measure{}, err
		}
	}

	// 24h>hole>1h, meter older than a week
	if !holeHour && meterWeek && holeDay {
		fmt.Println("CASE INTERPOLATION2")
		measure, err = insertLastWeekData(stub, operationID, meterID, consoProd, indexName, indexValue, lastMeasure, truncatedDate)
	}

	if !holeDay {
		fmt.Println("CASE LOST DATA")
		measure = sunchain.Measure{}
	}
	return measure, nil
}
