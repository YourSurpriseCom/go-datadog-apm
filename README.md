# Golang Datadog APM package with traces and a connected logger
[![Go Report Card](https://goreportcard.com/badge/github.com/YourSurpriseCom/go-datadog-apm)](https://goreportcard.com/report/github.com/YourSurpriseCom/go-datadog-apm) 
![workflow ci](https://github.com/YourSurpriseCom/go-datadog-apm/actions/workflows/ci.yml/badge.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# Description
This package makes it possible to add Datadog spans to your code and connect logs to your traces inside datadog based on the following parts:

* Datadog tracer
* Zap logger


## Usage
`go get github.com/YourSurpriseCom/go-datadog-apm`

```Go
import(
    github.com/YourSurpriseCom/go-datadog-apm/apm
)

apm := apm.NewApm()
currentContext := context.Background()

//Start a span
span, spanContext := apm.StartSpanFromContext(currentContext, "GetSignedUrl")
defer span.Finish()

//Create debug log message
apm.Logger.Debug(spanContext, "This log message will be linked to the span based on the spanContext")
```

## Configuration
The log level can be configure by setting the environment variable `LOG_LEVEL` with the following values:

* `debug`
* `info`
* `warning`
* `fatal`

When not set, it will fall back to the value `info`

Other custom logging options can be set by passing zap configurations to the logger constructor, and passing the custom logger to the apm constructor, like so:
```Go
import(
    github.com/YourSurpriseCom/go-datadog-apm/apm
    github.com/YourSurpriseCom/go-datadog-apm/logger
)

var myZapConfig *zap.Config // initialized elsewhere

logger := logger.NewLogger(
    WithConfig(myZapConfig)
)
return apm.NewApm(
    WithLogger(&logger)
)
```

## Serverless Config
To use the Serverless Datadog agent, build the application based on the following `Dockerfile`.

```Dockerfile
FROM    alpine:3.20
ARG     ARG_VERSION=dev
    
ENV     DD_VERSION=${ARG_VERSION}
ENV     DD_SITE=datadoghq.eu
    
COPY --from=datadog/serverless-init:1.2.8-alpine /datadog-init /datadog/datadog-init
COPY --from=builder /go/bin/main /go/bin/main

ENTRYPOINT ["/datadog/datadog-init"]

CMD ["/go/bin/main"]
```
