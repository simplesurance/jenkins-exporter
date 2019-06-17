# Jenkins Exporter #

An exporter for Jenkins Build metrics written in Golang.
The program is intended to run as daemon.
It fetches periodically metrics for Jenkins build via the Jenkins API and
publishes them via an HTTP endpoint in Prometheus format.

It provides the following Prometheus metrics:

- Summary Metric: `jenkins_exporter_job_duration_seconds`
  - Labels:
    - result
    - jenkins_job: the name of the Jenkins Job
    - type: type of recorded duration, one of:
      - blocked_time
      - buildable_time
      - building_duration
      - executing_time
      - waiting_time

By default metrics are recorded for every finished Jenkins build.
The jobs for that builds are recorded can be limited with the
`--jenkins-job-whitelist` command-line parameter.
The duration types that are recorded can also be configured via a commandline
parameter.
See `./jenkins-exporter -help` for more information.

## Installation ##

### Binary ###

#### go get ####

```
go get -u github.com/simplesurance/jenkins-exporter
```

#### git clone #####

```
git clone --depth 1 https://github.com/simplesurance/jenkins-exporter.git jenkins-exporter
cd jenkins-exporter
make
```

Then copy the jenkins-exporter into your `$PATH`.

### As Systemd service ###

1. Install `jenkins-exporter` to `/usr/local/bin`
2. `cp dist/etc/default/jenkins-exporter /etc/default/`
3. Configure the jenkins-exporter by editing `/etc/default/jenkins-exporter`
4. `cp dist/etc/systemd/system/jenkins-exporter.service /etc/systemd/system`
5. `systemctl daemon-reload && systemctl enable jenkins-exporter && systemctl start jenkins-exporter`
