package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

func recovery() {
	if r := recover(); r != nil {
		logReturn(r.(error), "Recovered from panic")
		return
	}
}

// logReturn formats the response when an action fails
func logReturn(err error, format string, args ...interface{}) pb.Response {
	defer timeTrack(time.Now(), "logReturn")
	if err == nil {
		err = errors.Errorf("ERROR : " + format + "err was nil, can't log the detail of it")
	}
	err = errors.Wrap(err, fmt.Sprintf("ERROR : "+format, args...))
	log.Print(err)
	return shim.Error(err.Error())
}

// logSuccess formats the response when an action succeeds
func logSuccess(payload []byte, format string, args ...interface{}) pb.Response {
	defer timeTrack(time.Now(), "logSuccess")
	if payload == nil {
		err := errors.Errorf("OK : "+format+"(there was no error in the query, and the payload was nil. Not good)", args...)
		return logReturn(err, "")
	}
	log.Printf("OK : "+format, args...)
	return shim.Success(payload)
}

// runeCP return the letter of the consoProd
func runeCP(consoProd string) rune {
	if consoProd == "Conso" {
		return 'C'
	}
	return 'P'
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
