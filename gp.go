package bo

import (
	"fmt"
	"math"

	"github.com/pkg/errors"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// GP represents a gaussian process.
type GP struct {
	inputs  [][]float64
	outputs []float64

	inputNames []string
	outputName string

	cov   Cov
	noise float64

	alpha        *mat.VecDense
	l            *mat.Cholesky
	mean, stddev float64

	n     int
	dirty bool
}

// NewGP creates a new Gaussian process with the specified covariance function
// (cov) and noise level (variance).
func NewGP(cov Cov, noise float64) *GP {
	return &GP{
		cov:   cov,
		noise: noise,
	}
}

// SetNames ...
func (gp *GP) SetNames(inputs []string, output string) {
	gp.inputNames = inputs
	gp.outputName = output
}

// Name ...
func (gp GP) Name(i int) string {
	if len(gp.inputNames) > i {
		name := gp.inputNames[i]
		if len(name) > 0 {
			return name
		}
	}
	return fmt.Sprintf("x[%d]", i)
}

// OutputName ...
func (gp GP) OutputName() string {
	if len(gp.outputName) > 0 {
		return gp.outputName
	}
	return "y"
}

// RawData ...
func (gp GP) RawData() ([][]float64, []float64) {
	inputs := make([][]float64, len(gp.inputs))
	for i, s := range gp.inputs {
		input := make([]float64, len(s))
		copy(input, s)
		inputs[i] = input
	}
	outputs := make([]float64, len(gp.outputs))
	copy(outputs, gp.outputs)
	return inputs, outputs
}

// Dims ...
func (gp GP) Dims() int {
	if len(gp.inputs) > 0 {
		return len(gp.inputs[0])
	}
	return 0
}

// Add bulk adds XY pairs.
func (gp *GP) Add(x []float64, y float64) {
	gp.dirty = true
	gp.inputs = append(gp.inputs, x)
	gp.outputs = append(gp.outputs, y)
}

func isConditionErr(err error) bool {
	_, ok := err.(mat.Condition)
	return ok
}

func (gp *GP) compute() error {
	defer func() {
		gp.dirty = false
	}()

	n := len(gp.inputs)
	k := mat.NewSymDense(n, nil)
	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			v := gp.cov.Cov(gp.inputs[i], gp.inputs[j])
			if i == j {
				v += gp.noise
			}
			k.SetSym(i, j, v)
		}
	}
	var L mat.Cholesky
	if ok := L.Factorize(k); !ok {
		return errors.Wrap(errors.New("failed to factorize"), "compute")
	}
	b := mat.NewVecDense(n, gp.normOutputs())
	alpha := mat.NewVecDense(n, nil)
	if err := L.SolveVecTo(alpha, b); err != nil && !isConditionErr(err) {
		return errors.Wrap(err, "failed to solve for alpha")
	}

	gp.alpha = alpha
	gp.l = &L
	gp.n = n
	return nil
}

func (gp *GP) normOutputs() []float64 {
	gp.mean, gp.stddev = stat.MeanStdDev(gp.outputs, nil)
	out := make([]float64, len(gp.outputs))
	for i, v := range gp.outputs {
		out[i] = (v - gp.mean) / gp.stddev
	}
	return out
}

// Estimate returns the mean and standard deviation at the point x.
func (gp *GP) Estimate(x []float64) (float64, float64, error) {
	if gp.dirty {
		if err := gp.compute(); err != nil {
			return 0, 0, errors.Wrap(err, "failed to run compute")
		}
	}
	n := gp.n

	kstar := mat.NewVecDense(n, nil)
	for i := 0; i < n; i++ {
		kstar.SetVec(i, gp.cov.Cov(gp.inputs[i], x))
	}
	mean := mat.Dot(kstar, gp.alpha)*gp.stddev + gp.mean

	v := mat.NewVecDense(n, nil)
	if err := gp.l.SolveVecTo(v, kstar); err != nil && !isConditionErr(err) {
		return 0, 0, errors.Wrap(err, "failed to find v")
	}
	variance := gp.cov.Cov(x, x) - mat.Dot(kstar, v)
	sd := math.Sqrt(variance)

	return mean, sd, nil
}

// Gradient returns the gradient of the mean at the point x.
func (gp *GP) Gradient(x []float64) ([]float64, error) {
	if gp.dirty {
		if err := gp.compute(); err != nil {
			return nil, errors.Wrap(err, "failed to run compute")
		}
	}
	n := gp.n

	kstar := mat.NewDense(len(x), n, nil)
	for i := 0; i < n; i++ {
		kstar.SetCol(i, gp.cov.Grad(gp.inputs[i], x))
	}

	grad := mat.NewVecDense(len(x), nil)
	grad.MulVec(kstar, gp.alpha)
	grad.ScaleVec(gp.stddev, grad)

	return grad.RawVector().Data, nil
}

// Minimum returns the minimum value logged.
func (gp *GP) Minimum() (x []float64, y float64) {
	i := floats.MinIdx(gp.outputs)
	return gp.inputs[i], gp.outputs[i]
}

// Maximum returns the maximum value logged.
func (gp *GP) Maximum() (x []float64, y float64) {
	i := floats.MaxIdx(gp.outputs)
	return gp.inputs[i], gp.outputs[i]
}

// Cov calculates the covariance between a and b.
type Cov interface {
	Cov(a, b []float64) float64
	Grad(a, b []float64) []float64
}

// MaternCov calculates the covariance between a and b. nu = 2.5
// https://en.wikipedia.org/wiki/Mat%C3%A9rn_covariance_function#Simplification_for_.CE.BD_half_integer
type MaternCov struct{}

// Cov ...
func (MaternCov) Cov(a, b []float64) float64 {
	const p = 2
	d := floats.Distance(a, b, 2)
	return (1 + math.Sqrt(5)*d/p + 5*d*d/(3*p*p)) * math.Exp(-math.Sqrt(5)*d/p)
}

// Grad computes the gradient of the matern covariance between a
// and b with respect to a. nu = 2.5.
func (MaternCov) Grad(a, b []float64) []float64 {
	d2 := floats.Distance(a, b, 2)
	d := make([]float64, len(a))
	floats.Add(d, a)
	floats.Sub(d, b)
	/*
		tmp := math.Sqrt(5 * floats.Sum(d))
		floats.Scale(5.0/3.0*(tmp+1)*math.Exp(-tmp), d)
	*/

	floats.Scale(math.Sqrt(5)+5.0/3.0*d2+math.Sqrt(5)*math.Exp(-math.Sqrt(5)/2.0*d2), d)
	return d
}
