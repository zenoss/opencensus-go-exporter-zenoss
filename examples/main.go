package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"

	zenoss "github.com/zenoss/opencensus-go-exporter-zenoss"
)

var (
	exampleCount = stats.Int64("example.com/measures/example.count", "example count", stats.UnitDimensionless)
	exampleSize  = stats.Int64("example.com/measures/example.size", "example size", stats.UnitBytes)
)

func main() {
	exporter, err := zenoss.NewExporter(zenoss.Options{})
	if err != nil {
		log.Fatal(err)
	}

	view.RegisterExporter(exporter)

	err = view.Register(
		&view.View{
			Name: "example.count",
			Description: "example count",
			Measure: exampleCount,
			Aggregation: view.Count(),
		},
		&view.View{
				Name: "example.size",
				Description: "example size",
				Measure: exampleSize,
				Aggregation: view.Distribution(0, 1<<16, 1<<32),
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	view.SetReportingPeriod(1 * time.Second)

	for {
		log.Print("looping...")
		stats.Record(context.Background(), exampleCount.M(1), exampleSize.M(rand.Int63()))
		<-time.After(time.Millisecond * time.Duration(1+rand.Intn(400)))
	}
}
