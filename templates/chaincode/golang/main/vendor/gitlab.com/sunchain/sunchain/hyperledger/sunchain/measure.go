package sunchain

import (
	"bytes"
	"fmt"
	"text/tabwriter"
	"time"
)

// Measure is a measure from a Meter
type Measure struct {
	IndexName    string    `json:"index_name"`
	IndexValue   int       `json:"index_value"`
	Timestamp    time.Time `json:"timestamp"`
	Delta        float64   `json:"delta"`
	Redistribute float64   `json:"redistribute"`
	Meter                  `json :"meter"`
}

// Allows the printing of a sunchain.Measure variable
func (m Measure) String() string {
	var b bytes.Buffer
	w := tabwriter.NewWriter(&b, 0, 0, 5, ' ', tabwriter.AlignRight)
	fmt.Fprintf(w, "📅 %s\toperation: %s\tmeter: %s\t consoProd : %s\tidx: %s\t⚡%d µWh\tΔ%d\t⏧%d µWh", m.Timestamp.Format(time.RFC3339), m.Meter.OperationID, m.Meter.ID, m.Meter.ConsoProd, m.IndexName, m.IndexValue, m.Delta, m.Redistribute)
	w.Flush()
	return b.String()
}
