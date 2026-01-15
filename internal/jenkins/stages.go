package jenkins

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type respWFAPIRaw struct {
	Stages []*respWFAPIStageRaw `json:"stages"`
}

type respWFAPIStageRaw struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	DurationMillis int64  `json:"durationMillis"`
}

// Stage holds metric data for a single jenkins build stage.
type Stage struct {
	Name     string
	Status   string
	Duration time.Duration
}

func respWFAPIRawToStage(raw *respWFAPIStageRaw) *Stage {
	return &Stage{
		Name:     raw.Name,
		Status:   raw.Status,
		Duration: time.Duration(raw.DurationMillis) * time.Millisecond,
	}
}

func respWFAPIRawToStages(resp *respWFAPIRaw) []*Stage {
	res := make([]*Stage, len(resp.Stages))
	for i, stage := range resp.Stages {
		res[i] = respWFAPIRawToStage(stage)
	}
	return res
}

func (c *Client) wfapiJobBuildURL(jobName, multibranchJobName, buildID string) (string, error) {
	if multibranchJobName == "" {
		return url.JoinPath(c.serverURL, "job", jobName, buildID, "wfapi")
	}

	return url.JoinPath(c.serverURL, "job", multibranchJobName, "job", jobName, buildID, "wfapi")
}

// Stages returns the metrics for all stages of a specific build.
func (c *Client) Stages(jobName, multibranchJobName string, buildID int64) ([]*Stage, error) {
	var resp respWFAPIRaw

	wfapiURL, err := c.wfapiJobBuildURL(jobName, multibranchJobName, fmt.Sprint(buildID))
	if err != nil {
		return nil, err
	}

	err = c.do(http.MethodGet, wfapiURL, &resp)
	if err != nil {
		return nil, err
	}

	return respWFAPIRawToStages(&resp), nil
}
