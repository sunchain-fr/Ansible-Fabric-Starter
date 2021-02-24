package sunchain

import (
	"fmt"
)

// Meter have the information of a meter
type Meter struct {
	ID          string `json:"meter_id"`
	ConsoProd   string `json:"conso_prod"`
	OperationID string `json:"op_id"`
}

//Allows the printing of a sunchain.Meter variable
func (m Meter) String() string {
	return fmt.Sprintf("MeterID : %s of type: %s belonging to the operation %s", m.ID, m.ConsoProd, m.OperationID)
}
