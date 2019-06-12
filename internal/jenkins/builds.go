package jenkins

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

//TODO: move to internal/

// https://github.com/jenkinsci/metrics-plugin/blob/master/src/main/java/jenkins/metrics/impl/TimeInQueueAction.java#L85
type actionRawResp struct {
	Class                  string `json:"_class"`
	WaitingTimeMillis      int64  `json:"waitingTimeMillis"`
	BuildableTimeMillis    int64  `json:"buildableTimeMillis"`
	BlockedTimeMillis      int64  `json:"blockedTimeMillis"`
	ExecutingTimeMillis    int64  `json:"executingTimeMillis"`
	BuildingDurationMillis int64  `json:"buildingDurationMillis"`
}
type buildRawResp struct {
	ID      string           `json:"id"`
	Actions []*actionRawResp `json:"actions"`
	Result  string           `json:"result"`
}

type jobRawResp struct {
	Name              string          `json:"name"`
	WorkflowJobBuilds []*buildRawResp `json:"builds"`
	MultiBranchJobs   []*jobRawResp   `json:"jobs"`
}

type respRaw struct {
	Jobs []*jobRawResp `json:"jobs"`
}

type Build struct {
	JobName            string
	MultibranchJobName string
	ID                 int64
	BuildableTime      time.Duration
	WaitingTime        time.Duration
	BlockedTime        time.Duration
	ExecutingTime      time.Duration
	BuildingDuration   time.Duration
	Result             string
}

func (c *Client) buildRawToBuild(workflowJobName, multibranchJobName string, rawBuild *buildRawResp) (*Build, error) {
	const metricClass = "jenkins.metrics.impl.TimeInQueueAction"

	for _, a := range rawBuild.Actions {
		if a.Class != metricClass {
			continue
		}

		intID, err := strconv.Atoi(rawBuild.ID)
		if err != nil {
			return nil, fmt.Errorf("could not convert id '%s' to int64", rawBuild.ID)
		}
		b := Build{
			JobName:            workflowJobName,
			MultibranchJobName: multibranchJobName,
			ID:                 int64(intID),
			BuildableTime:      time.Duration(a.BuildableTimeMillis) * time.Millisecond,
			WaitingTime:        time.Duration(a.WaitingTimeMillis) * time.Millisecond,
			BlockedTime:        time.Duration(a.BlockedTimeMillis) * time.Millisecond,
			ExecutingTime:      time.Duration(a.ExecutingTimeMillis) * time.Millisecond,
			BuildingDuration:   time.Duration(a.BuildingDurationMillis) * time.Millisecond,
			Result:             rawBuild.Result,
		}

		return &b, nil
	}

	return nil, errors.New("could not find metrics in Actions slice")
}

func buildIsInProgress(b *buildRawResp) bool {
	return b.Result == ""
}

func (c *Client) respRawToBuilds(raw *respRaw, removeInProgressBuilds bool) []*Build {
	var res []*Build

	for _, job := range raw.Jobs {
		for _, rawBuild := range job.WorkflowJobBuilds {
			if removeInProgressBuilds && buildIsInProgress(rawBuild) {
				continue
			}

			b, err := c.buildRawToBuild(job.Name, "", rawBuild)
			if err != nil {
				c.logger.Printf("skipping build %s/%s: %s", job.Name, rawBuild.ID, err)
				continue
			}

			res = append(res, b)
		}

		for _, multibranchJob := range job.MultiBranchJobs {
			for _, rawBuild := range multibranchJob.WorkflowJobBuilds {
				if removeInProgressBuilds && buildIsInProgress(rawBuild) {
					continue
				}

				b, err := c.buildRawToBuild(multibranchJob.Name, job.Name, rawBuild)
				if err != nil {
					c.logger.Printf("skipping build %s/%s/%s: %s", multibranchJob.Name, job.Name, rawBuild.ID, err)
					continue
				}

				res = append(res, b)
			}
		}
	}

	return res
}

func (c *Client) Builds(inProgressBuilds bool) ([]*Build, error) {
	// TODO: is it possible to retrieve only the element in actions with
	// _class = "jenkins.metrics.impl.TimeInQueueAction" that contains the
	// metrics?
	const queryBuilds = "builds[id,result,actions[_class,buildableTimeMillis,waitingTimeMillis,blockedTimeMillis,executingTimeMillis,buildingDurationMillis]]"
	const endpoint = "api/json" +
		"?tree=jobs[name,jobs[name," + queryBuilds + "]," + // multibranch jobs contain a job array that contain the builds
		queryBuilds + // workflow jobs contain the builds in the top-level job
		"]"

	var resp respRaw
	err := c.do("GET", c.serverURL+endpoint, &resp)
	if err != nil {
		return nil, err
	}

	builds := c.respRawToBuilds(&resp, !inProgressBuilds)

	return builds, nil
}
