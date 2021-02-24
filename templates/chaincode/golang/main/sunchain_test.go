package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"gitlab.com/sunchain/blockchain_kubernetes/artifacts/sunchain"
)

// To facilitate tests as now the chaincode truncates the time (15:39:49 -> 15:30:00) these times are created
var (
	testTime1 = time.Date(2017, 8, 3, 15, 31, 45, 569382419, time.UTC)
	testTime2 = time.Date(2017, 8, 3, 15, 31, 56, 569382419, time.UTC)
	testTime3 = time.Date(2017, 8, 3, 15, 32, 45, 569382419, time.UTC)
	testTime4 = time.Date(2017, 8, 3, 15, 33, 45, 569382419, time.UTC)
	testTime5 = time.Date(2017, 8, 3, 15, 34, 45, 569382419, time.UTC)
	testTime6 = time.Date(2017, 8, 3, 15, 35, 45, 569382419, time.UTC)
)

const (
	operationID1      = "ACC81T31234"
	operationID2      = "123481T3ACC"
	operationSpeciale = "PREMIAN_000"
	indexName1        = "IN1"
	indexName2        = "IN2"
	indexName3        = "OUT1"
	indexName4        = "OUT2"

	meterID1 = "11111111111111_11111"
	meterID2 = "22222222222222_22222"
	meterID3 = "33333333333333_33333"
	meterID4 = "44444444444444_44444"

	conso = "Conso"
	prod  = "Prod"
	c     = "C"
	p     = "P"
)

// check addMeterFlow errors and success
func TestAddMeter(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeterFlow(s, "yolo", meterID1, conso)
	addMeterFlow(s, operationID1, meterID1, "dollar")
	addMeterFlow(s, operationID1, "32786287462", conso)
	res := addMeterFlow(s, operationID1, meterID1, conso)

	var m sunchain.Meter
	json.Unmarshal(res.Payload, &m)
	if m.ID != meterID1 || m.ConsoProd != conso || m.OperationID != operationID1 {
		t.Fatalf("Wrong data inserted : %v", m)
	}

}

// check double insertion of the same meter
func TestAddConflictingMeter(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeterFlow(s, operationID1, meterID1, conso)
	addMeterFlow(s, operationID1, meterID1, conso)
	pb := getMetersFlow(s)
	var meters map[string]sunchain.Meter
	json.Unmarshal(pb.Payload, &meters)

	// If meters only has one member, we presume it worked
	if len(meters) != 1 {
		t.Fatalf("AddMeter created meters with the same ID : %v", meters)
	}
	if meters[meterID1+c].ID != meterID1 || meters[meterID1+c].OperationID != operationID1 || meters[meterID1+c].ConsoProd != conso {
		t.Fatalf("Wrong meter inserted")
	}
}

// check multiple meters insertion + check conso and prod for the same meter returns two different entries
func TestMeterID(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeterFlow(s, operationID1, meterID1, conso)
	addMeterFlow(s, operationID1, meterID1, prod)
	addMeterFlow(s, operationID2, meterID2, prod)
	r := getMetersFlow(s)

	var data map[string]sunchain.Meter
	json.Unmarshal(r.Payload, &data)

	if len(data) != 3 {
		t.Fatalf("Not all data has been inserted")
	}
	if data[meterID1+c].OperationID != operationID1 || data[meterID1+c].ID != meterID1 || data[meterID1+c].ConsoProd != conso {
		t.Fatalf("Meter1 was not saved in the blockchain.")
	}
	if data[meterID1+p].OperationID != operationID1 || data[meterID1+p].ID != meterID1 || data[meterID1+p].ConsoProd != prod {
		t.Fatalf("Meter2 was not saved in the blockchain.")
	}
	if data[meterID2+p].OperationID != operationID2 || data[meterID2+p].ID != meterID2 || data[meterID2+p].ConsoProd != prod {
		t.Fatalf("Meter3 wasn't saved in the blockchain")
	}

}

// check addMeasureFlow errors and success
func TestAddMeasure(t *testing.T) {
	var meters map[string]sunchain.Meter
	var measure sunchain.Measure
	empty := sunchain.Measure{}

	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID1+"NOPE", meterID1, conso, indexName1, 666, testTime1)
	addMeasureFlow(s, operationID1, meterID1+"NOPE", conso, indexName1, 666, testTime1)
	addMeasureFlow(s, operationID1, meterID1, conso+"NOPE", indexName1, 666, testTime1)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, -1000, testTime1)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 666, testTime4)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 666, testTime1)
	resMeter := getMetersFlow(s)

	resMeasure := getMeasureFlow(s, meterID1, testTime1, conso)
	json.Unmarshal(resMeasure.Payload, &measure)
	if measure != empty {
		t.Fatalf("Measure added before an existing measure -- should not be accepted")
	}

	resMeasure = getMeasureFlow(s, meterID1, testTime4, conso)
	json.Unmarshal(resMeasure.Payload, &measure)
	json.Unmarshal(resMeter.Payload, &meters)
	// we check if the meter created is right then if the measure returned at its creation
	// is equal to the one we get from the GetMeasures and finally, if the two have the right meter
	if len(meters) != 1 || meters[meterID1+c].ID != meterID1 || meters[meterID1+c].OperationID != operationID1 || meters[meterID1+c].ConsoProd != conso {
		t.Fatalf("Problem with meter insertion via addMeasure in the blockchain")
	}
	if measure.Meter.OperationID != operationID1 || measure.Meter.ID != meterID1 || measure.Meter.ConsoProd != conso || measure.IndexName != indexName1 || measure.IndexValue != 666 || measure.Timestamp != testTime4.Truncate(period) || measure.Delta != 0 || measure.Redistribute != 0 {
		t.Fatalf("Wrong measure inserted in the blockchain : %v", measure)
	}
}

// check insertion of different measures at the same date
func TestSameDateMeasures(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 666, testTime1)
	res := addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 200, testTime1)
	var m sunchain.Measure
	json.Unmarshal(res.Payload, &m)
	if m.IndexName != indexName1 || m.IndexValue != 200 || m.Delta != 0 {
		t.Fatalf("Wrong measure inserted in the blockchain : %v", m)
	}

	addMeasureFlow(s, operationID1, meterID1, conso, indexName2, 152, testTime1)
	res = getMeasureFlow(s, meterID1, testTime1, conso)

	json.Unmarshal(res.Payload, &m)
	if m.Meter.OperationID != operationID1 || m.Meter.ID != meterID1 || m.Meter.ConsoProd != conso || m.IndexName != indexName2 || m.IndexValue != 152 || m.Timestamp != testTime1.Truncate(period) || m.Delta != 0 || m.Redistribute != 0 {
		t.Fatalf("Wrong measure inserted in the blockchain : %v", m)
	}
}

// check delta calculation errors and results
func TestDeltaCalculation(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 666, testTime1)
	res := addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 669, testTime1)

	var measure sunchain.Measure
	json.Unmarshal(res.Payload, &measure)
	if measure.Delta != 0 {
		t.Fatalf("Delta calculated when it shouldn't : %v", measure)
	}

	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 777, testTime3)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 200, testTime3)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 1000, testTime3)
	res = getMeasureFlow(s, meterID1, testTime3, conso)

	json.Unmarshal(res.Payload, &measure)

	if measure.Delta != 331 {
		t.Fatalf("Delta calculation has failed : %v", measure)
	}
}

// check the result of an index change case
func TestIndexChange(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 100, testTime1)
	res := addMeasureFlow(s, operationID1, meterID1, conso, indexName2, 1000, testTime3)
	var m sunchain.Measure
	json.Unmarshal(res.Payload, &m)
	if m.IndexName != indexName2 || m.Delta != 0 {
		t.Fatalf("Wrong data insertion, delta calculation is false : %v", m)
	}
	res = addMeasureFlow(s, operationID1, meterID1, conso, indexName2, 2000, testTime4)
	json.Unmarshal(res.Payload, &m)
	if m.IndexName != indexName2 || m.Delta != 1000 {
		t.Fatalf("Wrong data insertion, delta calculation is false : %v", m)
	}
	res = addMeasureFlow(s, operationID1, meterID1, conso, indexName2, 2100, testTime5)
	json.Unmarshal(res.Payload, &m)
	if m.IndexName != indexName2 || m.Delta != 100 {
		t.Fatalf("Wrong data insertion, delta calculation is false : %v", m)
	}
	res = addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 120, testTime6)
	json.Unmarshal(res.Payload, &m)
	if m.IndexName != indexName1 || m.Delta != 20 {
		t.Fatalf("Wrong data insertion, delta calculation is false : %v", m)
	}
}

// check simple redistribution calculation
func TestRedistribute1_1(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 666, testTime1)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 999, testTime3)
	addMeasureFlow(s, operationID1, meterID2, prod, indexName2, 234, testTime1)
	addMeasureFlow(s, operationID1, meterID2, prod, indexName2, 543, testTime3)
	results := getMeasuresAtFlow(s, testTime3)

	var m map[string]sunchain.Measure
	json.Unmarshal(results.Payload, &m)

	if m[meterID2+p].Redistribute != 0 || m[meterID1+c].Redistribute != 309 {
		t.Fatalf("Redistribution is wrong")
	}
}

// Check redistribution calculation when more energy are consumed than produced and only one side has to calculate redistribution
func TestRedistributionC(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 666, testTime1)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 999, testTime3)
	addMeasureFlow(s, operationID1, meterID2, conso, indexName1, 111, testTime1)
	addMeasureFlow(s, operationID1, meterID2, conso, indexName1, 440, testTime3)
	addMeasureFlow(s, operationID1, meterID3, prod, indexName2, 234, testTime1)
	addMeasureFlow(s, operationID1, meterID3, prod, indexName2, 543, testTime3)

	results := getMeasuresAtFlow(s, testTime3)
	var m map[string]sunchain.Measure
	json.Unmarshal(results.Payload, &m)

	// numbers are the delta of each meter
	var redistributeMeter1, redistributeMeter2, totalProd float64

	// (333 / (333 + 329)) * 309
	redistributeMeter1 = 155

	// (329 / (333 + 329)) * 309
	redistributeMeter2 = 154

	totalProd = 309

	fmt.Println("1: ", redistributeMeter1, " 2: ", redistributeMeter2)
	if m[meterID1+c].Redistribute != redistributeMeter1 || m[meterID2+c].Redistribute != redistributeMeter2 || m[meterID3+p].Redistribute != 0 {
		t.Fatalf("Wrong redistribute calculation")
	}

	if (m[meterID1+c].Redistribute + m[meterID2+c].Redistribute + m[meterID3+p].Redistribute) != totalProd {
		t.Fatalf("Wrong redistribute calculation")
	}
}

//Check when more energy is produced than consumed with both sides calculating redistribution
func TestRedistributeP(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 2172439, testTime1)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 2172577, testTime3)
	addMeasureFlow(s, operationID1, meterID2, conso, indexName2, 578048, testTime1)
	addMeasureFlow(s, operationID1, meterID2, conso, indexName2, 578472, testTime3)
	addMeasureFlow(s, operationID1, meterID3, prod, indexName1, 492656, testTime1)
	addMeasureFlow(s, operationID1, meterID3, prod, indexName1, 493367, testTime3)
	addMeasureFlow(s, operationID1, meterID4, prod, indexName1, 347000, testTime1)
	addMeasureFlow(s, operationID1, meterID4, prod, indexName1, 347123, testTime3)

	res := getMeasuresAtFlow(s, testTime3)
	var measures map[string]sunchain.Measure
	json.Unmarshal(res.Payload, &measures)
	var totalProd, redistributeMeter1, redistributeMeter2, redistributeMeter3, redistributeMeter4 float64

	//deltaM1 := 138
	//deltaM2 := 424
	deltaM3 := 711.0
	deltaM4 := 123.0
	totalProd = deltaM3 + deltaM4

	redistributeMeter1 = 138
	redistributeMeter2 = 424
	redistributeMeter3 = 232
	redistributeMeter4 = 40

	for key, value := range measures {
		fmt.Println("TEST meterID :", key)
		fmt.Println("TEST value.Redistribute =", value.Redistribute)
	}

	fmt.Println("1: ", redistributeMeter1, " 2: ", redistributeMeter2, " 3: ", redistributeMeter3, " 4: ", redistributeMeter4)
	if measures[meterID4+p].Redistribute != redistributeMeter4 || measures[meterID3+p].Redistribute != redistributeMeter3 || measures[meterID1+c].Redistribute != redistributeMeter1 || measures[meterID2+c].Redistribute != redistributeMeter2 {
		fmt.Println("redistributeMeter1 expected =", redistributeMeter1, "got", measures[meterID1+c].Redistribute)
		fmt.Println("redistributeMeter2 expected =", redistributeMeter2, "got", measures[meterID2+c].Redistribute)
		fmt.Println("redistributeMeter3 expected =", redistributeMeter3, "got", measures[meterID3+p].Redistribute)
		fmt.Println("redistributeMeter4 expected =", redistributeMeter4, "got", measures[meterID4+p].Redistribute)
		t.Fatalf("Redistribution went wrong")
	}
	if (measures[meterID4+p].Redistribute + measures[meterID3+p].Redistribute + measures[meterID1+c].Redistribute + measures[meterID2+c].Redistribute) != totalProd {
		t.Fatalf("Redistribution went wrong.\nDifference between redistributed and total production = %d", (measures[meterID4+p].Redistribute + measures[meterID3+p].Redistribute + measures[meterID1+c].Redistribute + measures[meterID2+c].Redistribute) - totalProd)
	}
}

// check redistribution independant between operations
func TestRedistributeOperations(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID2, meterID1, conso, indexName1, 2172439, testTime1)
	addMeasureFlow(s, operationID2, meterID1, conso, indexName1, 2172577, testTime3)
	addMeasureFlow(s, operationID1, meterID2, conso, indexName2, 578048, testTime1)
	addMeasureFlow(s, operationID1, meterID2, conso, indexName2, 578472, testTime3)
	addMeasureFlow(s, operationID1, meterID3, prod, indexName1, 492656, testTime1)
	addMeasureFlow(s, operationID1, meterID3, prod, indexName1, 493367, testTime3)
	addMeasureFlow(s, operationID2, meterID4, prod, indexName1, 347000, testTime1)
	addMeasureFlow(s, operationID2, meterID4, prod, indexName1, 347123, testTime3)
	addMeasureFlow(s, operationID2, meterID4, conso, indexName1, 347123, testTime1)
	addMeasureFlow(s, operationID2, meterID4, conso, indexName1, 347150, testTime3)

	res := getMeasuresAtFlow(s, testTime3)
	var measures map[string]sunchain.Measure
	json.Unmarshal(res.Payload, &measures)
	var redistributeMeter1, redistributeMeter2, redistributeMeter3, redistributeMeter4c, redistributeMeter4p float64

	// op 1
	redistributeMeter2 = 424
	redistributeMeter3 = 287

	// op 2
	redistributeMeter1 = 103
	redistributeMeter4p = 0
	redistributeMeter4c = 20

	if measures[meterID4+p].Redistribute != redistributeMeter4p || measures[meterID3+p].Redistribute != redistributeMeter3 || measures[meterID1+c].Redistribute != redistributeMeter1 || measures[meterID2+c].Redistribute != redistributeMeter2 || measures[meterID4+c].Redistribute != redistributeMeter4c {
		t.Fatalf("Redistribution went wrong")
	}
}

// check getMeasuresBetween errors and results
func TestGetMeasuresBetween(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	getMeasuresBetweenFlow(s, testTime4, testTime4)
	getMeasuresBetweenFlow(s, testTime4, testTime1)
	getMeasuresBetweenFlow(s, testTime1, testTime4)

	addMeasureFlow(s, operationID1, meterID1, prod, indexName1, 666, testTime1)
	addMeasureFlow(s, operationID1, meterID1, prod, indexName1, 999, testTime3)
	addMeasureFlow(s, operationID1, meterID1, prod, indexName1, 1200, testTime4)
	addMeasureFlow(s, operationID1, meterID2, prod, indexName2, 888, testTime1)
	addMeasureFlow(s, operationID1, meterID2, prod, indexName2, 1000, testTime3)
	addMeasureFlow(s, operationID1, meterID2, prod, indexName2, 1523, testTime4)
	addMeasureFlow(s, operationID2, meterID3, conso, indexName2, 1000, testTime1)
	addMeasureFlow(s, operationID2, meterID3, conso, indexName2, 11222, testTime3)
	addMeasureFlow(s, operationID2, meterID3, conso, indexName2, 23147, testTime4)

	res := getMeasuresBetweenFlow(s, testTime1, testTime4)
	var m map[string][]sunchain.Measure
	json.Unmarshal(res.Payload, &m)

	if len(m) != 3 {
		t.Fatalf("Not all meters inserted")
	}
	for _, v := range m {
		if len(v) != 3 || v[0].IndexName != v[1].IndexName || v[0].ConsoProd != v[1].ConsoProd || v[0].OperationID != v[1].OperationID || v[0].ID != v[1].ID {
			t.Fatalf("Missing timestamp in the response ")
		}
	}
}

// check hole interpolation
func TestHoleInterpolation(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 130, testTime3)
	fmt.Println("first interpolation, hole < 10mn")
	testTime := testTime3.Add(9 * period)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 200, testTime)

	res := getMeasuresBetweenFlow(s, testTime3.Add(period), testTime.Add(-period))
	var m map[string][]sunchain.Measure
	json.Unmarshal(res.Payload, &m)

	delta := 8.0

	for _, v := range m {
		for _, vv := range v {
			if vv.Delta != delta {
				t.Fatalf("Wrong slope interpolation : %d != %d", vv.Delta, delta)
			}
		}
	}

	fmt.Println("first interpolation, hole<20mn but meter newer than 40mn")
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 500, testTime.Add(15*period))

	delta = 20
	res = getMeasuresBetweenFlow(s, testTime.Add(period), testTime.Add(14*period))
	json.Unmarshal(res.Payload, &m)
	for _, v := range m {
		for _, vv := range v {
			if vv.Delta != delta {
				t.Fatalf("Wrong slope interpolation : %d != %d", vv.Delta, delta)
			}
		}
	}
	fmt.Println("filling until meter 40mn old")
	testTime = testTime.Add(15 * period)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 700, testTime.Add(17*period))
	fmt.Println(testTime)
	delta = 12
	res = getMeasuresBetweenFlow(s, testTime.Add(period), testTime.Add(16*period))
	json.Unmarshal(res.Payload, &m)
	for _, v := range m {
		for _, vv := range v {
			if vv.Delta != delta {
				t.Fatalf("Wrong slope interpolation : %d != %d", vv.Delta, delta)
			}
		}
	}
	fmt.Println("first interpolation, hole < 10mn")
	testTime = testTime.Add(17 * period)
	fmt.Println(testTime)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 800, testTime.Add(9*period))
	res = getMeasuresBetweenFlow(s, testTime.Add(period), testTime.Add(8*period))
	json.Unmarshal(res.Payload, &m)
	delta = 11
	for _, v := range m {
		for _, vv := range v {
			if vv.Delta != delta {
				t.Fatalf("Wrong slope interpolation : %v != %d", vv, delta)
			}
		}
	}
	fmt.Println("second interpolation, hole < 20mn")
	testTime = testTime.Add(9 * period)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 1000, testTime.Add(18*period))
	// TODO : Find a way to check the results are good
	fmt.Println("hole > 20mn")
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 1400, testTime.Add(120*period))
	res = getMeasuresBetweenFlow(s, testTime.Add(period), testTime.Add(120*period))
	json.Unmarshal(res.Payload, &m)
	if len(m) != 1 {
		t.Fatalf("More than only one measure has been added ! : %v", m)
	}
	testTime = testTime.Add(120 * period)
	addMeasureFlow(s, operationID1, meterID1, conso, indexName1, 1500, testTime.Add(19*period))
	res = getMeasuresBetweenFlow(s, testTime.Add(period), testTime.Add(19*period))
	json.Unmarshal(res.Payload, &m)
	if len(m) != 1 {
		t.Fatalf("More than only one measure has been added ! : %v", m)
	}
}

func TestPackedDistribution(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationSpeciale, meterID1, conso, indexName1, 2172439, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID1, conso, indexName1, 2172577, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID1, prod, indexName1, 2172439, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID1, prod, indexName1, 2172787, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID2, conso, indexName2, 578048, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID2, conso, indexName2, 578130, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID3, conso, indexName1, 492656, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID3, conso, indexName1, 493054, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID4, prod, indexName1, 347000, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID4, prod, indexName1, 347155, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID4, conso, indexName1, 347123, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID4, conso, indexName1, 347143, testTime3)

	res := getMeasuresAtFlow(s, testTime3)
	var measures map[string]sunchain.Measure
	json.Unmarshal(res.Payload, &measures)
	var redistributeMeter1, redistributeMeter2, redistributeMeter3p, redistributeMeter3c, redistributeMeter4c, redistributeMeter4p float64

	redistributeMeter1 = 138
	redistributeMeter2 = 82
	redistributeMeter3p = 0
	redistributeMeter3c = 263
	redistributeMeter4p = 0
	redistributeMeter4c = 20

	for key, value := range measures {
		fmt.Println("TEST meterID :", key)
		fmt.Println("TEST value.Redistribute =", value.Redistribute)
	}

	if measures[meterID4+p].Redistribute != redistributeMeter4p || measures[meterID3+p].Redistribute != redistributeMeter3p || measures[meterID3+c].Redistribute != redistributeMeter3c || measures[meterID1+c].Redistribute != redistributeMeter1 || measures[meterID2+c].Redistribute != redistributeMeter2 || measures[meterID4+c].Redistribute != redistributeMeter4c {
		fmt.Println("redistributeMeter1 expected =", redistributeMeter1, "got", measures[meterID1+c].Redistribute)
		fmt.Println("redistributeMeter2 expected =", redistributeMeter2, "got", measures[meterID2+c].Redistribute)
		fmt.Println("redistributeMeter3p expected =", redistributeMeter3p, "got", measures[meterID3+p].Redistribute)
		fmt.Println("redistributeMeter3c expected =", redistributeMeter3c, "got", measures[meterID3+c].Redistribute)
		fmt.Println("redistributeMeter4p expected =", redistributeMeter4p, "got", measures[meterID4+p].Redistribute)
		fmt.Println("redistributeMeter4c expected =", redistributeMeter4c, "got", measures[meterID4+c].Redistribute)
		t.Fatalf("Redistribution went wrong")
	}
}

func TestRoundProduction(t *testing.T) {
	var totalProduction, totalConsumption, delta1, delta2 float64
	totalProduction = 2000
	totalConsumption = 1000

	var expected1, expected2 float64

	delta1 = 600
	expected1 = 300

	delta2 = 500
	expected2 = 250

	if roundProduction(totalConsumption, delta1, totalProduction) != expected1 || roundProduction(totalConsumption, delta2, totalProduction) != expected2 {
		fmt.Println("expected1 =", expected1, "got", roundProduction(totalConsumption, delta1, totalProduction))
		fmt.Println("expected2 =", expected1, "got", roundProduction(totalConsumption, delta2, totalProduction))
		t.Fatalf("RoundProduction went wrong")
	}
}

func TestPackedDistributionP(t *testing.T) {
	s := shim.NewMockStub("mockStub", &SunchainCode{})
	if s == nil {
		t.Fatalf("Mock Stub creation failed.")
	}
	s.MockTransactionStart("a1")
	defer s.MockTransactionEnd("a1")

	addMeasureFlow(s, operationSpeciale, meterID1, conso, indexName1, 2172439, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID1, conso, indexName1, 2172577, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID1, prod, indexName1, 2172439, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID1, prod, indexName1, 2172787, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID2, conso, indexName2, 578048, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID2, conso, indexName2, 578130, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID3, conso, indexName1, 492656, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID3, conso, indexName1, 493054, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID4, prod, indexName1, 347000, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID4, prod, indexName1, 347503, testTime3)
	addMeasureFlow(s, operationSpeciale, meterID4, conso, indexName1, 347123, testTime1)
	addMeasureFlow(s, operationSpeciale, meterID4, conso, indexName1, 347143, testTime3)

	res := getMeasuresAtFlow(s, testTime3)
	var measures map[string]sunchain.Measure
	json.Unmarshal(res.Payload, &measures)
	var redistributeMeter1c, redistributeMeter2, redistributeMeter1p, redistributeMeter3c, redistributeMeter4c, redistributeMeter4p float64

	redistributeMeter1c = 138
	redistributeMeter2 = 82
	redistributeMeter1p = 87
	redistributeMeter3c = 398
	redistributeMeter4p = 126
	redistributeMeter4c = 20

	for key, value := range measures {
		fmt.Println("TEST meterID :", key)
		fmt.Println("TEST value.Redistribute =", value.Redistribute)
	}

	if measures[meterID4+p].Redistribute != redistributeMeter4p || measures[meterID1+p].Redistribute != redistributeMeter1p || measures[meterID3+c].Redistribute != redistributeMeter3c || measures[meterID1+c].Redistribute != redistributeMeter1c || measures[meterID2+c].Redistribute != redistributeMeter2 || measures[meterID4+c].Redistribute != redistributeMeter4c {
		fmt.Println("redistributeMeter1c expected =", redistributeMeter1c, "got", measures[meterID1+c].Redistribute)
		fmt.Println("redistributeMeter2 expected =", redistributeMeter2, "got", measures[meterID2+c].Redistribute)
		fmt.Println("redistributeMeter1p expected =", redistributeMeter1p, "got", measures[meterID1+p].Redistribute)
		fmt.Println("redistributeMeter3c expected =", redistributeMeter3c, "got", measures[meterID3+c].Redistribute)
		fmt.Println("redistributeMeter4p expected =", redistributeMeter4p, "got", measures[meterID4+p].Redistribute)
		fmt.Println("redistributeMeter4c expected =", redistributeMeter4c, "got", measures[meterID4+c].Redistribute)
		t.Fatalf("Redistribution went wrong")
	}
}