# Jenkins Exporter #

An exporter for Jenkins Build metrics written in Golang.
The program fetches periodically metrics for Jenkins build via the Jenkins API
and publishes them via an HTTP endpoint in Prometheus format.

## Installation ##

### go get ##

```
go get github.com/simplesurance/jenkins-exporter
```

### git clone ###

```
git clone --depth 1 https://github.com/simplesurance/jenkins-exporter.git jenkins-exporter
cd jenkins-exporter
make
```

## Usage ## 

See `./jenkins-exporter -help`
