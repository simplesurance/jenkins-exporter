package main

import (
	_ "embed" // is used to initialize the version variable with content from the ver file
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jamiealquiza/envy"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/simplesurance/jenkins-exporter/internal/cli"
	"github.com/simplesurance/jenkins-exporter/internal/jenkins"
	"github.com/simplesurance/jenkins-exporter/internal/prometheus"
	"github.com/simplesurance/jenkins-exporter/internal/store"
)

const (
	appName = "jenkins-exporter"
)

// Version is set during compiliation via -ldflags.
var Version string = "unknown"

const stateStoreCleanupInterval = 10 * 60 * time.Second

const (
	blockedTimeMetricDescr  = "Time spent in the queue being blocked"
	buildAbleMetricDesc     = "Time spent in the queue while buildable"
	buildDurationMetricDesc = "Time from queuing to completion"
	executionTimeMetricDesc = "" // TODO: what is this metric
	waitingTimeMetricDesc   = "Time spent in the queue while waiting"
	stageDurationMetricDesc = "" // TODO: What does it show exactly? Runtime? Including queue time?
)

const buildStageMetricKey = "stage_duration_seconds"

var logger = log.New(os.Stderr, "", 0)
var debugLogger = log.New(io.Discard, "", 0)

var (
	listenAddr = flag.String("listen-addr", ":8123", "Listening address of the metric HTTP server")

	stateFilePath = flag.String("state-file", appName+".state.json", "path to the state file")
	maxStateAge   = flag.Duration("max-state-entry-age", 14*24*60*60*time.Second,
		"time after a state entry for a job is deleted if it was not updated")

	httpTimeout = flag.Duration("http-timeout", 180*time.Second, "Timeout for jenkins http requests")

	jenkinsUsername      = flag.String("jenkins-user", "", "Jenkins API username")
	jenkinsPassword      = flag.String("jenkins-password", "", "Jenkins API password or token")
	jenkinsURL           = flag.String("jenkins-url", "", "URL to the Jenkins Server")
	jenkinsJobWhitelist  = cli.StrMapFlag{}
	pollInterval         = flag.Duration("poll-interval", 30*time.Second, "Interval in which data is fetched from jenkins")
	httpRequestRateEvery = flag.Duration("http-request-limit", 500*time.Millisecond, "Limit the number of http-requests sent to jenkins to 1 per http-request-limit interval")

	prometheusNamespace = flag.String("prometheus-namespace", strings.ReplaceAll(appName, "-", "_"), "metric name prefix")

	recordBlockedTime      = flag.Bool("enable-blocked-time-metric", true, "record the blocked_time metric\n"+blockedTimeMetricDescr)
	recordBuildAbleTime    = flag.Bool("enable-buildable-time-metric", true, "record the buildable_time metric\n"+buildAbleMetricDesc)
	recordBuildingDuration = flag.Bool("enable-building-duration-metric", true, "record the building_duration metric\n"+buildDurationMetricDesc)
	recordExecutionTime    = flag.Bool("enable-execution-time-metric", false, "record the execution_time metric"+executionTimeMetricDesc) // no '\n' because executionTimeMetricDesc is an empty strinng
	recordWaitingTime      = flag.Bool("enable-waiting-time-metric", false, "record the waiting_time metric\n"+waitingTimeMetricDesc)
	histogramBuckets       = cli.Float64Slice{
		1 * 60,
		5 * 60,
		10 * 60,
		15 * 60,
		30 * 60,
		45 * 60,
		60 * 60,
		120 * 60,
		480 * 60,
	}

	recordBuildStageMetric        = flag.Bool("enable-build-stage-metric", false, "record the duration per build stages as a histogram metric called "+buildStageMetricKey)
	ignoreUnsuccessfulBuildStages = flag.Bool("build-stage-ignore-unsuccessful", true, "Ignore build stages that were unsuccessful.")
	recordBuildStageJobAllowList  = cli.BuildStageMapFlag{}

	printVersion = flag.Bool("version", false, "print the version and exit")
	debug        = flag.Bool("debug", false, "enable debug mode")
)

func init() {
	flag.Var(&jenkinsJobWhitelist, "jenkins-job-whitelist", "Comma-separated list of jenkins job names for that metrics are recorded.\nIf empty metrics for all jobs are recorded.\nMultibranch jobs are identified by their multibranch jobname.")
	flag.Var(&histogramBuckets, "histogram-buckets", "Comma-separated list of histogram buckets that are used for the metrics.\nValues are specified in seconds.")
	flag.Var(&recordBuildStageJobAllowList, "build-stage-allowlist", "Format: 'JOB-NAME[:STAGE-NAME][,JOB-NAME[:STAGE-NAME]]...'\nSpecifies jobs and stages for which per-stage metrics are recorded.\nIf empty, durations for all stages of all jobs are recorded.\nIf '[:STAGE_NAME]' is omitted, durations for all stages of the job are recorded.")
}

func recordJobDurationMetric(c *prometheus.Collector, jobName, metricType, buildResult, help string, duration time.Duration) {
	const key = "job_duration_seconds"
	c.Histogram(key, float64(duration/time.Second), help, histogramBuckets, map[string]string{
		// The label "job" is already used by Prometheus and
		// applied to all scrape targets.
		"jenkins_job": jobName,
		"type":        metricType,
		"result":      strings.ToLower(buildResult),
	})
}

// metricJobName returns the value of the job label used in metrics.
// If multibranchJobName is not empty, it is used as label value, otherwise
// jobName.
func metricJobName(b *jenkins.Build) string {
	if b.MultibranchJobName != "" {
		return b.MultibranchJobName
	}
	return b.JobName
}

func recordBuildMetric(c *prometheus.Collector, b *jenkins.Build) {
	jobName := metricJobName(b)

	// TODO: sanitize jobname?, sometimes contains %20 and other weird
	// chars

	if *recordBlockedTime {
		recordJobDurationMetric(c, jobName, "blocked_time", b.Result, blockedTimeMetricDescr, b.BlockedTime)
	}
	if *recordBuildAbleTime {
		recordJobDurationMetric(c, jobName, "buildable_time", b.Result, buildAbleMetricDesc, b.BuildableTime)
	}
	if *recordBuildingDuration {
		recordJobDurationMetric(c, jobName, "building_duration", b.Result, buildDurationMetricDesc, b.BuildingDuration)
	}
	if *recordExecutionTime {
		recordJobDurationMetric(c, jobName, "executing_time", b.Result, executionTimeMetricDesc, b.ExecutingTime)
	}
	if *recordWaitingTime {
		recordJobDurationMetric(c, jobName, "waiting_time", b.Result, waitingTimeMetricDesc, b.WaitingTime)
	}
}

func buildsByJob(builds []*jenkins.Build) map[string][]*jenkins.Build {
	res := map[string][]*jenkins.Build{}

	for _, b := range builds {
		var jobName string

		if b.MultibranchJobName != "" {
			jobName = b.MultibranchJobName + "/"
		}
		jobName += b.JobName

		jobBuilds := res[jobName]

		res[jobName] = append(jobBuilds, b)
	}

	return res
}

func sortDescByID(in map[string][]*jenkins.Build) {
	for _, builds := range in {
		sort.Slice(builds, func(i, j int) bool {
			return builds[i].ID > builds[j].ID
		})
	}
}

func jobIsInAllowList(build *jenkins.Build) bool {
	if len(jenkinsJobWhitelist) == 0 {
		return true
	}

	jobName := build.JobName
	if build.MultibranchJobName != "" {
		jobName = build.MultibranchJobName
	}

	if _, exist := jenkinsJobWhitelist[jobName]; exist {
		return true
	}

	return false
}

func recordBuildStageJobInAllowList(b *jenkins.Build) bool {
	if len(recordBuildStageJobAllowList) == 0 {
		return true
	}
	_, exists := recordBuildStageJobAllowList[metricJobName(b)]
	return exists
}

func fetchAndRecord(clt *jenkins.Client, store *store.Store, onlyRecordNewbuilds bool, collector *prometheus.Collector) error {
	fetchStart := time.Now()

	builds, err := clt.Builds(false)
	if err != nil {
		return err
	}

	logger.Printf("retrieved %d builds from jenkins in %s", len(builds), time.Since(fetchStart))

	buildMap := buildsByJob(builds)
	sortDescByID(buildMap)

	for job, builds := range buildMap {
		if len(builds) == 0 {
			continue
		}

		// We can not pass job here because it's in the format
		// <MultiBranchJobName>/<JobName>. The whitelist contains
		// either the MultiBranchJobName or the JobName
		// TODO: would be more efficient to only retrieve the information for the jobs
		// that we are interesting in from the API, instead of retrieving all
		// and ignoring some
		recordJobMetrics := jobIsInAllowList(builds[0])
		recordperStageMetrics := *recordBuildStageMetric && recordBuildStageJobInAllowList(builds[0])
		if !recordJobMetrics && !recordperStageMetrics {
			continue
		}

		highestID, exist := store.Get(job)
		if !exist && onlyRecordNewbuilds {
			// If a new state file was created and we do not have
			// a record for the build, do not record metrics for
			// builds that already existed in the first iteration.
			// On a subsequent runs new builds will be recorded.
			// This prevents that we record multiple times the
			// same builds if the state store file of the the
			// previous execution was deleted but metrics for jobs
			// were recorded in Prometheus.

			store.Set(job, builds[0].ID)
			debugLogger.Printf("%s: seen the first time and state file did not exist, skipping existing builds, highest build ID: %d", job, builds[0].ID)
			continue
		}
		if highestID > builds[0].ID {
			debugLogger.Printf("%s: highest stored job ID is bigger then the one known by jenkins, resetting ID to: %d", job, builds[0].ID)
			store.Set(job, builds[0].ID)
		}

		for _, b := range builds {
			if b.ID <= highestID {
				// all following builds are known
				break
			}

			if recordJobMetrics {
				recordBuildMetric(collector, b)
			}
			if recordperStageMetrics {
				fetchAndRecordStageMetric(clt, collector, b)
			}
		}
		if builds[0].ID > highestID {
			logger.Printf("%s: recorded metrics for %d build(s), new highest build ID: %d", job, builds[0].ID-highestID, builds[0].ID)
			debugLogger.Printf("%s: highest seen build ID is %d", job, builds[0].ID)
			store.Set(job, builds[0].ID)
		}
	}

	return nil
}

func fetchAndRecordStageMetric(clt *jenkins.Client, collector *prometheus.Collector, b *jenkins.Build) {
	stages, err := clt.Stages(b.JobName, b.MultibranchJobName, b.ID)
	if err != nil {
		logger.Printf("retrieving stage information for job: %q, multibranchJob: %q, buildID: %d, failed: %s", b.JobName, b.MultibranchJobName, b.ID, err)
		collector.CounterAdd("errors", 1, "jenkins api fetch errors", map[string]string{"type": "jenkins_wfapi"})
		return
	}

	recordStagesMetric(collector, b, stages)
}

func stageIsInAllowList(jobName, stageName string) bool {
	stageMap := recordBuildStageJobAllowList[jobName]
	if len(stageMap) == 0 {
		return true
	}
	_, exists := stageMap[stageName]
	return exists
}

func recordStagesMetric(c *prometheus.Collector, b *jenkins.Build, stages []*jenkins.Stage) {
	metricJobName := metricJobName(b)
	for _, stage := range stages {
		if !stageIsInAllowList(metricJobName, stage.Name) {
			continue
		}

		if *ignoreUnsuccessfulBuildStages && !strings.EqualFold(stage.Status, "success") {
			continue
		}

		c.Histogram(buildStageMetricKey, float64(stage.Duration/time.Second), stageDurationMetricDesc, histogramBuckets, map[string]string{
			"jenkins_job": metricJobName,
			"stage":       stage.Name,
			"type":        "duration",
			"result":      strings.ToLower(stage.Status),
		})
	}
}

func loadOrCreateStateStore() (isNewStore bool, _ *store.Store) {
	stateStore, err := store.FromFile(*stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Printf("state file '%s' does not exist", *stateFilePath)
			return true, store.New()
		}

		logger.Fatalf("loading state file failed: %s", err)
	}

	logger.Printf("state loaded from '%s'", *stateFilePath)

	return false, stateStore
}

func registerSigHandler(s *store.Store) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan

		logger.Printf("received %s signal, terminating...", sig)

		err := s.ToFile(*stateFilePath)
		if err != nil {
			logger.Printf("saving statefile failed: %s", err)
		} else {
			logger.Printf("state written to %s", *stateFilePath)
		}

		os.Exit(0)
	}()
}

func logConfiguration() {
	const fmtSpec = "%-40s: %v\n"

	str := "Configuration:\n"
	str += fmt.Sprintf(fmtSpec, "Jenkins URL", *jenkinsURL)
	str += fmt.Sprintf(fmtSpec, "Jenkins Username", *jenkinsUsername)
	str += fmt.Sprintf(fmtSpec, "Jenkins Password", "##hidden##")
	str += fmt.Sprintf(fmtSpec, "Jenkins Job Whitelist", jenkinsJobWhitelist.String())
	str += fmt.Sprintf(fmtSpec, "Listen Address", *listenAddr)
	str += fmt.Sprintf(fmtSpec, "State File", *stateFilePath)
	str += fmt.Sprintf(fmtSpec, "Max State Entry Age", maxStateAge.String())
	str += fmt.Sprintf(fmtSpec, "Poll Interval", pollInterval.String())
	str += fmt.Sprintf(fmtSpec, "HTTP Request Limit, 1 per", httpRequestRateEvery.String())
	str += fmt.Sprintf(fmtSpec, "HTTP Timeout (sec)", httpTimeout.String())
	str += fmt.Sprintf(fmtSpec, "Prometheus Namespace", *prometheusNamespace)
	str += fmt.Sprintf(fmtSpec, "Histogram Buckets", histogramBuckets.String())
	str += fmt.Sprintf(fmtSpec, "Record blocked_time", *recordBlockedTime)
	str += fmt.Sprintf(fmtSpec, "Record buildable_time", *recordBuildAbleTime)
	str += fmt.Sprintf(fmtSpec, "Record building_duration", *recordBuildingDuration)
	str += fmt.Sprintf(fmtSpec, "Record execution_time", *recordExecutionTime)
	str += fmt.Sprintf(fmtSpec, "Record waiting_time", *recordWaitingTime)
	str += fmt.Sprintf(fmtSpec, "Record per Stage Metrics", *recordBuildStageMetric)
	str += fmt.Sprintf(fmtSpec, "Ignore Unsuccessful Build Stages", *ignoreUnsuccessfulBuildStages)
	str += fmt.Sprintf(fmtSpec, "Build Stage Allowlist", recordBuildStageJobAllowList.String())

	logger.Printf(str)
}

func validateFlags() {
	if *jenkinsURL == "" {
		fmt.Printf("Error: -jenkins-url parameter must be specified\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	envy.Parse("JE")
	flag.Parse()

	if *printVersion {
		fmt.Printf("%s\n", strings.TrimSpace(Version))
		os.Exit(0)
	}

	validateFlags()

	if *debug {
		debugLogger.SetOutput(os.Stderr)
	}

	logConfiguration()

	http.Handle("/", promhttp.Handler())
	go func() {
		logger.Printf("prometheus http server listening on %s", *listenAddr)
		err := http.ListenAndServe(*listenAddr, nil)
		if err != http.ErrServerClosed {
			logger.Fatal("prometheus http server terminated:", err.Error())
		}

	}()

	isNewStore, stateStore := loadOrCreateStateStore()
	registerSigHandler(stateStore)

	collector := prometheus.NewCollector(*prometheusNamespace, nil)
	clt := jenkins.NewClient(*jenkinsURL).
		WithAuth(*jenkinsUsername, *jenkinsPassword).
		WithLogger(debugLogger).
		WithTimeout(*httpTimeout).
		WithMetrics(collector).
		WithRatelimit(*httpRequestRateEvery)

	nextStateStoreCleanup := time.Now()

	// set error counters to 0
	collector.CounterAdd("errors", 0, "jenkins api fetch errors", map[string]string{"type": "jenkins_api"})
	collector.CounterAdd("errors", 0, "jenkins api fetch errors", map[string]string{"type": "jenkins_wfapi"})

	for {
		err := fetchAndRecord(clt, stateStore, isNewStore, collector)
		if err != nil {
			logger.Printf("fetching and recording builds metrics failed: %s", err)
			collector.CounterAdd("errors", 1, "jenkins api fetch errors", map[string]string{"type": "jenkins_api"})
		}

		if nextStateStoreCleanup.Before(time.Now()) {
			cnt := stateStore.RemoveOldEntries(*maxStateAge)
			nextStateStoreCleanup = time.Now().Add(stateStoreCleanupInterval)
			logger.Printf("removed %d expired entries from the state store, next cleanup in %s", cnt, stateStoreCleanupInterval)
		}

		logger.Printf("fetching and recording the next build metrics in %s", *pollInterval)
		time.Sleep(*pollInterval)
	}
}
