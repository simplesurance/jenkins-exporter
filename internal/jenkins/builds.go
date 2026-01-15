// Package jenkins implements a client for the jenkins API.
package jenkins

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

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
	ID       string           `json:"id"`
	Actions  []*actionRawResp `json:"actions"`
	Result   string           `json:"result"`
	Building *bool            `json:"building"`
}

type jobRawResp struct {
	Name              string          `json:"name"`
	WorkflowJobBuilds []*buildRawResp `json:"builds"`
	MultiBranchJobs   []*jobRawResp   `json:"jobs"`
}

type respRaw struct {
	Jobs []*jobRawResp `json:"jobs"`
}

// Build holds metric data for a single jenkins build.
type Build struct {
	JobName            string
	MultiBranchJobName string
	ID                 int64
	BuildableTime      time.Duration
	WaitingTime        time.Duration
	BlockedTime        time.Duration
	ExecutingTime      time.Duration
	BuildingDuration   time.Duration
	Result             string
	Building           bool
}

// FullJobName returns the full job name, including the multibranch job name if present.
func (b *Build) FullJobName() string {
	if b.MultiBranchJobName != "" {
		return b.MultiBranchJobName + "/" + b.JobName
	}
	return b.JobName
}

func (b *Build) String() string {
	return b.FullJobName() + " #" + fmt.Sprint(b.ID)
}

func (b *buildRawResp) validate() error {
	if b.Building == nil {
		return errors.New("building field is missing (nil)")
	}

	if !*b.Building && b.Result == "" {
		return errors.New("building is false but build result is empty")
	}

	return nil
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
			MultiBranchJobName: multibranchJobName,
			ID:                 int64(intID),
			BuildableTime:      time.Duration(a.BuildableTimeMillis) * time.Millisecond,
			WaitingTime:        time.Duration(a.WaitingTimeMillis) * time.Millisecond,
			BlockedTime:        time.Duration(a.BlockedTimeMillis) * time.Millisecond,
			ExecutingTime:      time.Duration(a.ExecutingTimeMillis) * time.Millisecond,
			BuildingDuration:   time.Duration(a.BuildingDurationMillis) * time.Millisecond,
			Result:             rawBuild.Result,
			Building:           *rawBuild.Building,
		}

		return &b, nil
	}

	return nil, errors.New("could not find metrics in Actions slice")
}

func buildID(job *jobRawResp, build *buildRawResp) string {
	return fmt.Sprintf("%s/%s", job.Name, build.ID)
}
func multibranchBuildID(multibranchJob, job *jobRawResp, build *buildRawResp) string {
	return fmt.Sprintf("%s/%s/%s", multibranchJob.Name, job.Name, build.ID)
}

func (c *Client) respRawToBuilds(raw *respRaw) []*Build {
	var res []*Build

	for _, job := range raw.Jobs {
		for _, rawBuild := range job.WorkflowJobBuilds {
			if err := rawBuild.validate(); err != nil {
				c.logger.Printf("skipping build %s: %s", buildID(job, rawBuild), err)
				continue
			}

			b, err := c.buildRawToBuild(job.Name, "", rawBuild)
			if err != nil {
				c.logger.Printf("skipping build %s: %s", buildID(job, rawBuild), err)
				continue
			}

			res = append(res, b)
		}

		for _, multibranchJob := range job.MultiBranchJobs {
			for _, rawBuild := range multibranchJob.WorkflowJobBuilds {
				if err := rawBuild.validate(); err != nil {
					c.logger.Printf("skipping build %s: %s", multibranchBuildID(multibranchJob, job, rawBuild), err)
					continue
				}

				b, err := c.buildRawToBuild(multibranchJob.Name, job.Name, rawBuild)
				if err != nil {
					c.logger.Printf("skipping build %s: %s", multibranchBuildID(multibranchJob, job, rawBuild), err)
					continue
				}

				res = append(res, b)
			}
		}
	}

	return res
}

// Builds retrieves a list of all builds and their metrics from the jenkins server.
func (c *Client) Builds() ([]*Build, error) {
	// TODO: is it possible to retrieve only the element in actions with
	// _class = "jenkins.metrics.impl.TimeInQueueAction" that contains the
	// metrics?
	const queryBuilds = "builds[id,result,building,actions[_class,buildableTimeMillis,waitingTimeMillis,blockedTimeMillis,executingTimeMillis,buildingDurationMillis]]"
	const endpoint = "api/json" +
		"?tree=jobs[name,jobs[name," + queryBuilds + "]," + // multibranch jobs contain a job array that contain the builds
		queryBuilds + // workflow jobs contain the builds in the top-level job
		"]"

	var resp respRaw
	err := c.do("GET", c.serverURL+endpoint, &resp)
	if err != nil {
		return nil, err
	}

	builds := c.respRawToBuilds(&resp)

	return builds, nil
}
