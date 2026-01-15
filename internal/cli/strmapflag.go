package cli

import "strings"

// StrMapFlag is a flag.Value implementation for string maps.
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

// Set implements the flag.Value interface.
func (s *StrMapFlag) Set(v string) error {
	*s = StrMapFlag{}

	for _, elem := range strings.Split(v, ",") {
		(*s)[strings.TrimSpace(elem)] = struct{}{}
	}

	return nil
}
