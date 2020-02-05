package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	zenoss "github.com/zenoss/opencensus-go-exporter-zenoss"
)

var (
	serviceTagKey = tag.MustNewKey("service")

	exampleCount = stats.Int64("example.com/measures/example.count", "example count", stats.UnitDimensionless)
	exampleSize  = stats.Int64("example.com/measures/example.size", "example size", stats.UnitBytes)
)

func main() {
	apiKey := os.Getenv("ZENOSS_API_KEY")
	if apiKey == "" {
		// API key is mandatory.
		log.Fatal("ZENOSS_API_KEY environment variable must be set.")
	}

	source := os.Getenv("ZENOSS_SOURCE")
	if source == "" {
		// Source is mandatory.
		log.Fatal("ZENOSS_SOURCE environment variable must be set.")
	}

	exporterOptions := zenoss.Options{
		APIKey: apiKey,

		// GlobalDimensions are added to all sent metrics and models.
		GlobalDimensions: map[string]string{"source": source},

		// ModelDimensionTags selects OpenCensus stats tags to use as Zenoss dimensions.
		ModelDimensionTags: []string{serviceTagKey.Name()},

		// Metric names matching these regular expressions won't be sent to Zenoss.
		IgnoredMetricNames: []string{},
	}

	// Only override address if it's set in the environment.
	if zenossAddress := os.Getenv("ZENOSS_ADDRESS"); zenossAddress != "" {
		exporterOptions.Address = zenossAddress
	}

	exporter, err := zenoss.NewExporter(exporterOptions)
	if err != nil {
		log.Fatal(err)
	}

	view.RegisterExporter(exporter)

	err = view.Register(
		&view.View{
			Name:        "example.count",
			Description: "example count",
			Measure:     exampleCount,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{serviceTagKey},
		},
		&view.View{
			Name:        "example.size",
			Description: "example size",
			Measure:     exampleSize,
			Aggregation: view.Distribution(0, 1<<16, 1<<32),
			TagKeys:     []tag.Key{serviceTagKey},
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	view.SetReportingPeriod(1 * time.Minute)

	// Add our tag to the context we'll use to record stats.
	ctx, _ := tag.New(context.Background(), tag.Insert(serviceTagKey, "example"))

	log.Print("recording stats...")
	for {
		stats.Record(ctx, exampleCount.M(1), exampleSize.M(rand.Int63()))
		<-time.After(time.Millisecond * time.Duration(1+rand.Intn(1000)))
	}
}
