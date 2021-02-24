package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"gitlab.com/sunchain/sunchain/hyperledger/sunchain"
	)

//deltaCalculation calculates the delta between each measure
func deltaCalculation(old, new sunchain.Measure) (float64, error) {
	defer timeTrack(time.Now(), "deltaCalculation")
	var delta float64

	if new.IndexValue < old.IndexValue {
		return 0, fmt.Errorf("negative production or consumption in the last period for (%v,%v). %v -> %v", old.Timestamp, new.Meter.ID, old.IndexValue, new.IndexValue)
	}

	if new.IndexName == old.IndexName && new.Timestamp == old.Timestamp {
		delta = round(old.Delta+float64(new.IndexValue-old.IndexValue), 0.1)
	} else if new.IndexName == old.IndexName {
		delta = round(float64(new.IndexValue-old.IndexValue), 0.1)
	}

	return delta, nil
}

// For the second redistribution strategy, we pack meter in ordered batch :
//  - each batch can contain 1..n meters
//  - for each ordered batch, we do a classic redistribution
//  - if there is some energy left after a "batch redistribution",
//  - we redo the classic redistribution on the next batch
func packedRedistributionCalculation(meters map[string]sunchain.Meter, measures map[string]sunchain.Measure, operationID string) (map[string]sunchain.Measure, error) {
	defer timeTrack(time.Now(), "packedRedistributionCalculation")
	_, _, totalProduction := deltaSum(meters, measures, operationID)
	packedMeters, err := sunchain.GetMeterInPack(operationID)
	if err != nil {
		return nil, err
	}
	// first of all, we calculate the redistributed part for prod meters
	measures = redistributionCalculationProdMeter(meters, measures, operationID)
	// then we calculate the redistribution part for each sub pack
	for i := 0; i < len(packedMeters); i++ {
		// We select meters in the current pack
		tmpMesure := make(map[string]sunchain.Measure)
		metersID := packedMeters[i+1]
		metersPack := make(map[string]sunchain.Meter)
		measuresPack := make(map[string]sunchain.Measure)
		var curr time.Time
		for _, packMeterID := range metersID {
			updatatedMeterID := packMeterID + "C"
			metersPack[updatatedMeterID] = meters[updatatedMeterID]
			measuresPack[updatatedMeterID] = measures[updatatedMeterID]
			curr = measuresPack[updatatedMeterID].Timestamp
		}
		if len(metersPack) == 0 || len(measuresPack) == 0 {
			log.Println("something went wrong while calculating at", curr.Format(time.RFC3339)+". len(metersPack) =", len(metersPack), "\tlen(measuresPack) =", len(measuresPack))
		}
		log.Println("total production for pack number", i+1, "=", totalProduction)
		tmpMesure, totalProduction = redistributionCalculationPackedMeters(metersPack, measuresPack, operationID, totalProduction)
		for meterID, measure := range tmpMesure {
			measures[meterID] = measure
		}
	}
	return measures, nil
}

func redistributionCalculationProdMeter(meters map[string]sunchain.Meter, measures map[string]sunchain.Measure, operationID string) map[string]sunchain.Measure {
	_, totalConsumption, productionTotal := deltaSum(meters, measures, operationID)
	for meterID, meter := range meters {
		if meter.OperationID != operationID {
			continue
		}
		if _, ok := measures[meterID]; ok && meter.ConsoProd == "Prod" && productionTotal > 0 {
			redistributed := roundProduction(totalConsumption, measures[meterID].Delta, productionTotal)
			tmpMeasure := measures[meterID]
			if tmpMeasure.ID == "" || tmpMeasure.IndexName == "" || tmpMeasure.OperationID == "" {
				log.Println("measure stocked in measures map smees to be nil. tmpMesure :", tmpMeasure.String(), "\tmeasures[meterID] :", measures[meterID].String())
			} else {
				tmpMeasure.Redistribute = redistributed
				measures[meterID] = tmpMeasure
			}
		}
	}
	return measures
}

func redistributionCalculationPackedMeters(meters map[string]sunchain.Meter, measures map[string]sunchain.Measure, operationID string, totalProduction float64) (map[string]sunchain.Measure, float64) {
	if totalProduction < 0 {
		return measures, 0
	}
	var totalReditributed float64
	var timerDEBUG time.Time
	totalReditributed = 0
	_, totalConsumption, _ := deltaSum(meters, measures, operationID)
	for meterID, meter := range meters {
		measure := measures[meterID]
		if meter.ConsoProd == "Prod" || meter.OperationID != operationID {
			continue
		}
		measure.Redistribute = roundConsumption(totalProduction, measure.Delta, totalConsumption)
		totalReditributed += measure.Redistribute
		measures[meterID] = measure
		timerDEBUG = measure.Timestamp
	}
	log.Println("sub total consumption at", timerDEBUG.Format(time.RFC3339), "=", totalConsumption)
	productionRest := totalProduction - totalReditributed
	if productionRest < 0 {
		productionRest = 0
	}
	return measures, round(productionRest, 0.1)
}

//redistributionCalculation calculate the redistributed part for each meter
func redistributionCalculation(meters map[string]sunchain.Meter, measures map[string]sunchain.Measure, operationID string) map[string]sunchain.Measure {
	defer timeTrack(time.Now(), "redistributionCalculation")
	totalDistribution, consumptionTotal, productionTotal := deltaSum(meters, measures, operationID)

	for meterID, meter := range meters {
		if meter.OperationID != operationID {
			continue
		}
		m := measures[meterID]
		if meter.ConsoProd == "Conso" && consumptionTotal > 0 {
			m.Redistribute = roundConsumption(totalDistribution, m.Delta, consumptionTotal)
		}
		if meter.ConsoProd == "Prod" && productionTotal > 0 {
			m.Redistribute = roundProduction(consumptionTotal, m.Delta, productionTotal)
		}
		measures[meterID] = m
	}
	return measures
}

//deltaSum calculates for a specific operation the total of conso, prod and the max of them
func deltaSum(meters map[string]sunchain.Meter, measures map[string]sunchain.Measure, operationID string) (float64, float64, float64) {
	defer timeTrack(time.Now(), "deltaSum")

	var (
		consumptionTotal  float64
		productionTotal   float64
		totalDistribution float64
	)

	for meterID, meter := range meters {
		if meter.OperationID != operationID {
			continue
		}
		if meter.ConsoProd == "Conso" {
			consumptionTotal += measures[meterID].Delta
		} else {
			productionTotal += measures[meterID].Delta
		}
	}

	if consumptionTotal > productionTotal {
		totalDistribution = productionTotal
	} else {
		totalDistribution = consumptionTotal
	}
	return totalDistribution, consumptionTotal, productionTotal
}

func round(x, unit float64) float64 {
	return float64(int64(x/unit+0.5)) * unit
}

// roundConsumption calculates the redistribution and round it to 0.1 ( 7.2124 = 7.2, 7.6784 = 7.7)
func roundConsumption(totalProduction, delta, totalConsumption float64) float64 {
	defer timeTrack(time.Now(), "roundConsumption")
	if totalProduction > totalConsumption {
		return round(delta, 0.1)
	}
	if totalProduction == 0 {
		return 0.0
	}
	return round((delta/totalConsumption)*totalProduction, 0.1)

}

// roundProduction calculates the production's redistribution surplus and round it
func roundProduction(totalConsumption, delta, totalProduction float64) float64 {
	defer timeTrack(time.Now(), "roundProduction")
	if totalProduction < totalConsumption {
		return 0.0
	}
	return round((totalProduction-totalConsumption)*(delta/totalProduction), 0.1)

}

// discretePeriod returns all periods between two timestamps
func discretePeriod(start, end time.Time) []time.Time {
	defer timeTrack(time.Now(), "discretePeriod")
	interval := make([]time.Time, 0)
	for curr := start; curr.Before(end.Add(period)); curr = curr.Add(period) {
		interval = append(interval, curr)
	}
	return interval
}

func linearInterpolation(stub shim.ChaincodeStubInterface, operationID, meterID, consoProd, indexName string, indexValue int, lastMeasure sunchain.Measure, truncatedDate time.Time) (sunchain.Measure, error) {
	defer timeTrack(time.Now(), "linearInterpolation")
	m := lastMeasure
	var err error

	times := discretePeriod(lastMeasure.Timestamp, truncatedDate)
	slope := int(math.Floor((float64(indexValue)-float64(m.IndexValue))/float64(len(times)-1) + 0.5))
	times = times[1 : len(times)-1]
	for _, v := range times {
		m, err = addSingleMeasure(stub, operationID, meterID, consoProd, indexName, m.IndexValue+slope, v, m)
		if err != nil {
			return sunchain.Measure{}, err
		}
		log.Println("INTERPOLATED VALUE 1:", m)
	}
	return m, nil
}

func ratioCalculation(iVnow, iVlast, iVold2, iVold1 int) (float64, error) {
	defer timeTrack(time.Now(), "ratioCalculation")
	if iVold2-iVold1 <= 0 {
		return 0, fmt.Errorf("ratio calculation is wrong : %d <= %d", iVold2, iVold1)
	}
	return float64(iVnow-iVlast) / float64(iVold2-iVold1), nil
}

//insertLastWeekData recreates measures adapting last week's deltas.
// 1. Comparison the delta difference between this week's missing measures and last week's missing measure.
// 2. Calculate ratio accordingly.
// 3. Create missing measures with delta = last week's measures's delta * ratio
func insertLastWeekData(stub shim.ChaincodeStubInterface, operationID, meterID, consoProd, indexName string, indexValue int, lastMeasure sunchain.Measure, truncatedDate time.Time) (sunchain.Measure, error) {
	defer timeTrack(time.Now(), "insertLastWeekData")
	m := lastMeasure
	// we get the missing timestamps, and the measures before and after the hole and the ones of the last week
	times := discretePeriod(lastMeasure.Timestamp.Add(period), truncatedDate)
	oldm1, err := getMeasure(stub, meterID, lastMeasure.Timestamp.Add(-time.Duration(40*time.Minute)), consoProd)
	if err != nil {
		return sunchain.Measure{}, err
	}
	oldm2, err := getMeasure(stub, meterID, truncatedDate.Add(-time.Duration(40*time.Minute)), consoProd)
	if err != nil {
		return sunchain.Measure{}, err
	}

	// if there is no data on one of this timestamp, we simply add the measure
	if oldm1.Timestamp.IsZero() || oldm2.Timestamp.IsZero() {
		m = sunchain.Measure{}
		m, err = addSingleMeasure(stub, operationID, meterID, consoProd, indexName, indexValue, truncatedDate, m)
		if err != nil {
			return sunchain.Measure{}, err
		}
		return m, nil
	}

	// we calculated the ratio
	var ratio, interpolatedValue float64
	ratio, err = ratioCalculation(indexValue, m.IndexValue, oldm2.IndexValue, oldm1.IndexValue)
	if err != nil {
		return sunchain.Measure{}, err
	}

	// for each missing timestamp, we add the value based on the last week's delta times the ratio
	var lastWeekMeasure sunchain.Measure
	times = times[:len(times)-1]
	for _, v := range times {
		lastWeekMeasure, err = getMeasure(stub, meterID, v.Add(-time.Duration(40*time.Minute)), consoProd)
		if err != nil {
			return sunchain.Measure{}, err
		}
		interpolatedValue = float64(m.IndexValue) + float64(lastWeekMeasure.Delta)*ratio
		m, err = addSingleMeasure(stub, operationID, meterID, consoProd, indexName, int(interpolatedValue), v, m)
		if err != nil {
			return sunchain.Measure{}, err
		}
		log.Println("INTERPOLATED VALUE 2:", m)
	}
	return m, nil
}
