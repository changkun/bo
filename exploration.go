package bo

import (
	"math"
)

// Exploration is the strategy to use for exploring the Gaussian process.
type Exploration interface {
	Estimate(gp *GP, minimize bool, x []float64) (float64, error)
}

// UCB implements upper confidence bound exploration.
type UCB struct {
	Kappa float64
}

// Estimate implements Exploration.
func (e UCB) Estimate(gp *GP, minimize bool, x []float64) (float64, error) {
	mean, sd, err := gp.Estimate(x)
	if err != nil {
		return 0, err
	}
	if minimize {
		return mean - e.Kappa*sd, nil
	}
	return mean + e.Kappa*sd, nil
}

type EI struct {
}

func (e EI) Estimate(gp *GP, minimize bool, x []float64) (float64, error) {
	mean, std, err := gp.Estimate(x)
	if err != nil {
		return 0, err
	}
	a := mean //  (mean - y_max)? ymax?
	z := a / std

	if minimize {
		return a*StdNormal.CDF(z) - std*StdNormal.PDF(z), nil
	}

	return a*StdNormal.CDF(z) + std*StdNormal.PDF(z), nil
}

// NormalDist is a normal (Gaussian) distribution with mean Mu and
// standard deviation Sigma.
type NormalDist struct {
	Mu, Sigma float64
}

// StdNormal is the standard normal distribution (Mu = 0, Sigma = 1)
var StdNormal = NormalDist{0, 1}

// 1/sqrt(2 * pi)
const invSqrt2Pi = 0.39894228040143267793994605993438186847585863116493465766592583

func (n NormalDist) PDF(x float64) float64 {
	z := x - n.Mu
	return math.Exp(-z*z/(2*n.Sigma*n.Sigma)) * invSqrt2Pi / n.Sigma
}

func (n NormalDist) CDF(x float64) float64 {
	return math.Erfc(-(x-n.Mu)/(n.Sigma*math.Sqrt2)) / 2
}

// BarrierFunc returns a value that is added to the value to bound the
// optimization.
type BarrierFunc interface {
	Val(x []float64, params []Param) float64
	Grad(x []float64, params []Param) []float64
}

// BasicBarrier returns -Inf if an x value is outside the param range.
func BasicBarrier(x []float64, params []Param) float64 {
	for i, p := range params {
		v := x[i]
		if v < p.GetMin() || v > p.GetMax() {
			return math.Inf(-1)
		}
	}
	return 0
}

// LogBarrier implements a logarithmic barrier function.
type LogBarrier struct{}

// Val returns the value of the barrier function.
func (LogBarrier) Val(x []float64, params []Param) float64 {
	v := 0.0
	for i, p := range params {
		v += math.Log2(p.GetMax() - x[i])
		v += math.Log2(x[i] - p.GetMin())
	}
	if math.IsNaN(v) {
		return math.Inf(-1)
	}
	return v
}

// Grad returns the gradient of the barrier function.
func (LogBarrier) Grad(x []float64, params []Param) []float64 {
	grad := make([]float64, len(x))
	for i, p := range params {
		grad[i] = 1/(x[i]-p.GetMin()) - 1/(p.GetMax()-x[i])
		// TODO: handle NaN
	}
	return grad
}
