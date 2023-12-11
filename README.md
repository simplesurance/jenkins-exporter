# Jenkins Exporter #

An exporter for Jenkins Build metrics written in Golang.
The program is intended to run as daemon.
It fetches periodically metrics for Jenkins builds and Stages via the Jenkins
API and publishes them via an HTTP endpoint in Prometheus format.

It provides the following Prometheus metrics:

- Histograms:
  - `jenkins_exporter_job_duration_seconds_bucket`,  
    `jenkins_exporter_job_duration_seconds_sum`,  
    `jenkins_exporter_job_duration_seconds_count`  
    - Labels:
      - result
      - jenkins_job: the name of the Jenkins Job
      - branch: for multibranch Jenkins Jobs, the branch name
    - type: type of recorded duration, one of:
      - blocked_time
      - buildable_time
      - building_duration
      - executing_time
      - waiting_time
  - `jenkins_exporter_stage_duration_seconds_bucket`,  
    `jenkins_exporter_stage_duration_seconds_sum`,  
    `jenkins_exporter_stage_duration_seconds_count`,  
    - Labels:
      - result: result of the stage
      - stage: stage name
      - type:
        - duration
      - branch: for multibranch Jenkins Jobs, the branch name
- Counter:
  - `jenkins_exporter_errors`  
    Counts the number of errors the jenkins-exporter encountered when fetching
    information from the Jenkins API.
    - Labels:
      - type:
        - jenkins_api
        - jenkins_wfapi

By default job metrics are recorded for every finished Jenkins build. The
Jenkins jobs for that builds are recorded can be limited with the
`--jenkins-job-whitelist` command-line parameter.
The duration types that are recorded can also be configured via a command-line
parameter.

The exporter can also record durations of individual pipeline stages. This
requires that the
[Pipeline Stage View Plugin](https://plugins.jenkins.io/pipeline-rest-api/) is
installed on the Jenkins server. Recording per stage metrics can be enabled via the
`-enable-build-stage-metrics` command-line parameter. The
`-build-stage-allowlist` parameter allows to specify for which jobs and stages,
per-stage metrics are recorded.

All parameter can also be specified via environment variables.
See `./jenkins-exporter -help` for more information.

## Installation ##

### Binary ###

#### Download Release Binary ####

Download a release binary from: <https://github.com/simplesurance/jenkins-exporter/releases>.

#### go install ####

```sh
go install github.com/simplesurance/jenkins-exporter@latest
```

#### git clone #####

```sh
git clone --depth 1 https://github.com/simplesurance/jenkins-exporter.git jenkins-exporter
cd jenkins-exporter
make
```

Then copy the jenkins-exporter into your `$PATH`.

### As Systemd Service ###

1. Install `jenkins-exporter` to `/usr/local/bin`
2. `cp dist/etc/default/jenkins-exporter /etc/default/`
3. Configure the jenkins-exporter by editing `/etc/default/jenkins-exporter`
4. `cp dist/etc/systemd/system/jenkins-exporter.service /etc/systemd/system`
5. `systemctl daemon-reload && systemctl enable jenkins-exporter && systemctl start jenkins-exporter`
