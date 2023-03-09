package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type StrMapFlag map[string]struct{}

func (s *StrMapFlag) String() string {
	var str string
	var i int

	for k := range *s {
		str += k
		if i < len(k)-1 {
			str += ", "
		}

		i++
	}

	return str
}

func (s *StrMapFlag) Set(v string) error {
	*s = StrMapFlag{}

	for _, elem := range strings.Split(v, ",") {
		(*s)[strings.TrimSpace(elem)] = struct{}{}
	}

	return nil
}

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

// BuildStageMapFlag parses a a string in the format:
// "<JOB-NAME>:<STAGE-NAME>,..."
// into a map[JOB-NAME]map[STAGE-NAME]struct{} datastructure.
// If an elem ends with a "," it is interpreted as being on the Job-Name.
// Later elemens overwrite earlier elements.
type BuildStageMapFlag map[string]map[string]struct{}

func (b BuildStageMapFlag) Set(v string) error {
	for _, elem := range strings.Split(v, ",") {
		idx := strings.LastIndex(elem, ":")
		if idx == -1 || len(elem) == idx+1 {
			b[elem] = map[string]struct{}{}
			continue
		}

		job := elem[:idx]
		stage := elem[idx+1:]
		if m, exists := b[job]; exists {
			m[stage] = struct{}{}
			continue
		}

		b[job] = map[string]struct{}{stage: {}}
	}

	return nil
}

func (b BuildStageMapFlag) String() string {
	res := strings.Builder{}

	for job, stageMap := range b {
		if len(stageMap) == 0 {
			res.WriteString(job)
			res.WriteRune(',')
			continue
		}
		for stage := range stageMap {
			res.WriteString(job)
			res.WriteRune(':')
			res.WriteString(stage)
			res.WriteRune(',')
		}
	}

	return strings.TrimRight(res.String(), ",")

}
