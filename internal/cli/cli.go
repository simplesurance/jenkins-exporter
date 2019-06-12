package cli

import (
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
	for _, elem := range strings.Split(v, ",") {
		(*s)[strings.TrimSpace(elem)] = struct{}{}
	}

	return nil
}
