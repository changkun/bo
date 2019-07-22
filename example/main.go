package main

import (
	"log"
	"math"

	"github.com/changkun/bo"
)

func main() {
	X := bo.UniformParam{
		Max: 10,
		Min: -10,
	}
	o := bo.NewOptimizer(
		[]bo.Param{
			X,
		},
	)
	// minimize x^2+1
	x, y, err := o.Run(func(params map[bo.Param]float64) float64 {
		return math.Pow(params[X], 2) + 1
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(x, y)
}
