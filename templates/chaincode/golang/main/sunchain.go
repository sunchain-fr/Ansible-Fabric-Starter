package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//meterKey and measureIndex enables to select all meters/measures from the state
const (
	meterKey     = "meter:"
	measureIndex = "measure:"
	period       = 10 * time.Minute
)

// SunchainCode is a dummy chaincode struct to be able to implement the shim interface
type SunchainCode struct{}

func (t *SunchainCode) Init(stub shim.ChaincodeStubInterface) (resp pb.Response) {
	defer recovery()
	fmt.Println("Entering the pool")
	resp = shim.Success(nil)
	return resp
}

//API to chaincode functions.
// Verifies for each functionnality the correct number of inputs before calling the functions
// and transforms timestamps to time.Time
func (t *SunchainCode) Invoke(stub shim.ChaincodeStubInterface) (resp pb.Response) {
	defer timeTrack(time.Now(), "Invoke")
	defer recovery()
	function, args := stub.GetFunctionAndParameters()
	switch function {
	case "AddMeter":
		if len(args) != 3 {
			return shim.Error("Parameters awaited by addMeter : operationID /string meterID /string and consoProd /string (must be Conso or Prod) ")
		}
		operationID := args[0]
		meterID := args[1]
		consoProd := args[2]
		return addMeterFlow(stub, operationID, meterID, consoProd)

	case "AddMeasure":
		if len(args) != 6 {
			return shim.Error("Parameters awaited by addMeasure : operationID /string meterID /string, consoProd /string (must be Conso or Prod), indexName /string, indexValue /uint, timestamp /string (format example: 2008-09-08T22:47:31+01:00)")
		}
		operationID := args[0]
		meterID := args[1]
		consoProd := args[2]
		nomIndex := args[3]
		valIndex, _ := strconv.Atoi(args[4])
		timestamp := args[5]
		ts, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return logReturn(err, "Couldn't parse timestamp into time")
		}
		return addMeasureFlow(stub, operationID, meterID, consoProd, nomIndex, valIndex, ts)

	case "GetMeters":
		return getMetersFlow(stub)

	case "GetMeasure":
		if len(args) != 3 {
			return shim.Error("Parameters awaited by GetMeasure : meterID /string and timestamp /string (RFC3339 format, eg: 2008-09-08T22:47:31+01:00)") //TODO: ajouter le troisième argument ConsoProd
		}
		meterID := args[0]
		timestamp := args[1]
		ts, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return logReturn(err, "Couldn't parse timestamp into time")
		}
		consoProd := args[2] //TODO: if it doesn't involve changing too much stuff everywhere, change the order of args for consistency between functions
		return getMeasureFlow(stub, meterID, ts, consoProd)

	case "GetMeasuresAndRedistribute":
		if len(args) != 1 {
			return shim.Error("Parameters awaited by GetMeasuresAndRedistribute : timestamp /string (RFC3339 format, eg: 2008-09-08T22:47:31+01:00)")
		}
		timestamp := args[0]
		ts, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return logReturn(err, "Couldn't parse timestamp into time")
		}
		return getMeasuresAtFlow(stub, ts)

	case "GetMeasuresBetween":
		if len(args) != 2 {
			return shim.Error("Parameters awaited by GetMeasuresBetween : start /string and end /string (RFC3339 format, eg: 2008-09-08T22:47:31+01:00)")
		}
		start := args[0]
		st, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return logReturn(err, "Couldn't parse starting timestamp into time")
		}
		end := args[1]
		nd, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return logReturn(err, "Couldn't parse ending timestamp into time")
		}
		return getMeasuresBetweenFlow(stub, st, nd)

	}
	return shim.Error("Unknown action")
}

func main() {
	// prevents chaincode failing due to an error
	defer recovery()
	err := shim.Start(&SunchainCode{})
	if err != nil {
		log.Println(err)
	}
}
