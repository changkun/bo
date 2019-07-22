package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/aybabtme/rgbterm"
	"github.com/changkun/bo"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	X := bo.UniformParam{
		Max:  1,
		Min:  0,
		Name: "X",
	}
	Y := bo.UniformParam{
		Max:  1,
		Min:  0,
		Name: "Y",
	}
	Z := bo.UniformParam{
		Max:  1,
		Min:  0,
		Name: "Z",
	}
	o := bo.NewOptimizer(
		[]bo.Param{
			X, Y, Z,
		},
		bo.WithMinimize(false),
		bo.WithRounds(10),
	)
	x, y, err := o.RunSerial(func(params map[bo.Param]float64) float64 {
		for {
			r, g, b := uint8(params[X]*255), uint8(params[Y]*255), uint8(params[Z]*255)
			word := "██████"
			coloredWord := rgbterm.FgString(word, r, g, b)
			fmt.Println("Is this your prefered color? ", coloredWord, " (", r, ", ", g, ", ", b, ")")
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Grade? (0 dislike to 5 like):")
			text, _ := reader.ReadString('\n')
			res, err := strconv.ParseFloat(text[:len(text)-1], 64)
			if err != nil {
				fmt.Println("err: ", err)
			}
			fmt.Println("res: ", res)
			if res >= 0 && res <= 5 {
				return res
			}
			fmt.Println("Wrong input, please input a number between 0 ~ 5!")
		}
	})
	if err != nil {
		log.Fatal(err)
	}
	r, g, b := uint8(x[X]*255), uint8(x[Y]*255), uint8(x[Z]*255)
	word := "██████"
	coloredWord := rgbterm.FgString(word, r, g, b)
	fmt.Println("prefered color: ", coloredWord, " (", r, ", ", g, ", ", b, ")", " score: ", y)
}
