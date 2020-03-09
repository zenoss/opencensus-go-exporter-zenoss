# Zenoss OpenCensus Go Exporter

[![GoDoc][godoc-image]][godoc-url]

_opencensus-go-exporter-zenoss_ is a [Go] library intended to be used in [Go]
applications instrumented with [OpenCensus] to export stats to [Zenoss].

[Go]: https://golang.org/
[OpenCensus]: https://opencensus.io/
[Zenoss]: https://zenoss.com/

[godoc-image]: https://godoc.org/github.com/zenoss/opencensus-go-exporter-zenoss?status.svg
[godoc-url]: https://godoc.org/github.com/zenoss/opencensus-go-exporter-zenoss

## Installation

You can install this library into your GOPATH with the following command.

```shell script
go get -u github.com/zenoss/opencensus-go-exporter-zenoss
```

## Usage

Using the exporter requires importing the package, creating an exporter, and
registering the exporter.

```go
package main

import (
	// OpenCensus packages.
	"go.opencensus.io/stats/view"

	// Zenoss helper for sending directly to a single API endpoint.
	"github.com/zenoss/zenoss-go-sdk/endpoint"

	// Zenoss OpenCensus exporter.
	"github.com/zenoss/opencensus-go-exporter-zenoss/zenoss"
)

const (
    ZenossAPIKey = "YOUR-API-KEY"
    ZenossSource = "YOUR-APPLICATION"
)

func main() {
	// Create a Zenoss API client to which we'll send metrics. 
	client, _ := endpoint.New(endpoint.Config{APIKey: ZenossAPIKey})

    // Create Zenoss OpenCensus exporter.
    exporter, _ := zenoss.NewExporter(zenoss.Options{
        Output: client,
        Source: ZenossSource,
    })

    // Register Zenoss OpenCensus exporter. 
    view.RegisterExporter(exporter)

    // Instrument your application with OpenCensus stats.

    // Flush client before exiting to send any buffered metrics.
    client.Flush()
}
```

A complete working example can be found in the [examples/] directory.

[examples/]: https://github.com/zenoss/opencensus-go-exporter-zenoss/tree/master/examples
