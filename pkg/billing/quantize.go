package billing

import (
	"math"

	"github.com/shopspring/decimal"
)

// MonetaryScale defines the 8-decimal scale aligned with sub2api (NUMERIC(20,8))
const MonetaryScale = 8

// QuantizeAmount quantizes a monetary amount to 8 decimal places using
// half-away-from-zero rounding, exactly matching sub2api and PostgreSQL NUMERIC.
func QuantizeAmount(v float64) float64 {
	if v == 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}
	quantized, _ := decimal.NewFromFloat(v).Round(MonetaryScale).Float64()
	return quantized
}
