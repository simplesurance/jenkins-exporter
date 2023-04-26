package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type Float64Slice []float64

func (u *Float64Slice) String() string {
	var str string
	var i int

	for _, v := range *u {
		str += fmt.Sprint(v)
		if i < len(*u)-1 {
			str += ", "
		}

		i++
	}

	return str
}

func (u *Float64Slice) Set(v string) error {
	*u = nil

	strVals := strings.Split(v, ",")
	for _, v := range strVals {
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return err
		}
		*u = append(*u, f)
	}

	return nil
}
