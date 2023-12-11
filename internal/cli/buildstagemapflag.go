package cli

import "strings"

// BuildStageMapFlag parses a a string in the format:
// "<JOB-NAME>:<STAGE-NAME>,..."
// into a map[JOB-NAME]map[STAGE-NAME]struct{} datastructure.
// If an elem ends with a "," it is interpreted as being on the Job-Name.
// Later element overwrite earlier elements.
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
