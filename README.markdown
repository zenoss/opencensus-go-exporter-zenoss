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

```
go get -u github.com/zenoss/opencensus-go-exporter-zenoss
```

## Usage

Using the exporter requires importing the package, creating an exporter, and
registering the exporter.

```go
package main

import (
    "go.opencensus.io/stats/view"
    zenoss "github.com/zenoss/opencensus-go-exporter-zenoss"
)

func main() {
    options := zenoss.Options{
        APIKey: "YOUR-ZENOSS-API-KEY",

        // GlobalDimensions are added to all sent metrics and models.
        GlobalDimensions: map[string]string{"source": "YOUR-APP-NAME"},

        // ModelDimensionTags selects OpenCensus stats tags to use as Zenoss dimensions.
        ModelDimensionTags: []string{"grpc_server_method", "grpc_server_status"},
    }

    exporter, err := zenoss.NewExporter(options)
    if err != nil {
        panic(err)
    }

    view.RegisterExporter(exporter)

    // Instrument your application with OpenCensus stats.
}
```

A complete working example can be found in the [examples/] directory.

[examples/]: https://github.com/zenoss/opencensus-go-exporter-zenoss/tree/master/examples
