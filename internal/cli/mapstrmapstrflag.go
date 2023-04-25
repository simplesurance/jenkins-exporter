package cli

import (
	"fmt"
	"strings"
)

// MapStrMapStrFlag parses a flag value in the format:
// 'KEY:val[,val...][;KEY:val[,val...]... into a map[string]map[string]struct{}
type MapStrMapStrFlag map[string]map[string]struct{}

func (s MapStrMapStrFlag) String() string {
	var result strings.Builder
	var i int

	for k, v := range s {
		var valCnt int
		result.WriteString(k)
		result.WriteRune(':')

		for val := range v {
			result.WriteString(val)
			if valCnt < len(v)-1 {
				result.WriteRune(',')
			}

			valCnt++
		}

		if i < len(s)-1 {
			result.WriteRune(';')
		}

		i++
	}

	return result.String()
}

func (s MapStrMapStrFlag) Set(v string) error {
	for _, elem := range strings.Split(v, ";") {
		jobname, strbranchNames, found := strings.Cut(elem, ":")
		if !found {
			return fmt.Errorf("':' separator not found in substring '%s'", elem)
		}

		branchNames := strings.FieldsFunc(strbranchNames, func(r rune) bool { return r == ',' })
		fmt.Println(branchNames)
		s[jobname] = strSliceToMap(branchNames)
	}

	return nil
}

func strSliceToMap(sl []string) map[string]struct{} {
	result := make(map[string]struct{}, len(sl))
	for _, elem := range sl {
		result[elem] = struct{}{}
	}

	return result
}
