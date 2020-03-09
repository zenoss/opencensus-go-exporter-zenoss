package zenoss

import (
	"context"
	"fmt"

	"go.opencensus.io/stats/view"

	"github.com/zenoss/zenoss-protobufs/go/cloud/data_receiver"
)

const (
	// SourceField is the tag identifying a metric's source.
	SourceField = "source"

	// SourceTypeField is a tag identifying the type of source.
	SourceTypeField = "source-type"

	// DefaultSourceType is the default source-type identifying this OpenCensus exporter.
	DefaultSourceType = "zenoss/opencensus-go-exporter-zenoss"

	// DescriptionField is the optional tag containing a metric's description.
	DescriptionField = "description"

	// UnitsField is the optional tag containing a metric's unit of measure.
	UnitsField = "units"
)

var (
	// Ensure we implement view.Explorer interface.
	_ view.Exporter = (*Exporter)(nil)
)

// Options defines the options for a Zenoss OpenCensus exporter.
type Options struct {
	// Output is anything that implements Zenoss' DataReceiverServiceClient.
	// See NewEndpoint, NewSplitter, and NewProcessor in zenoss/zenoss-go-sdk.
	// Required.
	Output data_receiver.DataReceiverServiceClient

	// Source is a Zenoss tag commonly used to identify the source of data.
	// Recommended.
	Source string

	// SourceType is a Zenoss tag commonly used to identify the type of source.
	// Optional. (Default: zenoss/opencensus-go-exporter-zenoss)
	SourceType string

	// ExtraTags is a map of extra tags to add to every metric.
	ExtraTags map[string]string
}

type Exporter struct {
	options Options
	output  data_receiver.DataReceiverServiceClient
}

// NewExporter returns a new Zenoss OpenCensus exporter.
func NewExporter(options Options) (*Exporter, error) {
	e := &Exporter{}
	if err := e.SetOptions(options); err != nil {
		return nil, err
	}

	return e, nil
}

// SetOutput allows changing the exporter's options even while it's running.
func (e *Exporter) SetOptions(options Options) error {
	if options.Output == nil {
		return fmt.Errorf("zenoss.Options.Output must be specified")
	}

	if options.SourceType == "" {
		options.SourceType = DefaultSourceType
	}

	e.options = options
	e.output = options.Output

	return nil
}

// Flush waits for exported data to be sent. Call before program termination.
func (e *Exporter) Flush() {
	i, ok := e.output.(interface{ Flush() })
	if ok {
		i.Flush()
	}
}

// ExportView exports stats to Zenoss. Satisfies view.Exporter interface.
func (e *Exporter) ExportView(viewData *view.Data) {
	metrics := make([]*data_receiver.TaggedMetric, 0, 5)
	timestamp := viewData.End.UnixNano() / 1e6

	addMetric := func(name string, value float64, tags map[string]string) {
		description := viewData.View.Description
		if description != "" {
			tags[DescriptionField] = description
		}

		units := viewData.View.Measure.Unit()
		if units != "" {
			tags[UnitsField] = units
		}

		metrics = append(metrics, &data_receiver.TaggedMetric{
			Metric:    name,
			Timestamp: timestamp,
			Value:     value,
			Tags:      tags,
		})
	}

	for _, viewRow := range viewData.Rows {
		tags := make(map[string]string)

		for _, rowTag := range viewRow.Tags {
			tags[rowTag.Key.Name()] = rowTag.Value
		}

		// Copy ExtraTags. Overwrite metric tags of the same name.
		for k, v := range e.options.ExtraTags {
			tags[k] = v
		}

		// Set Source. Overwrite metric tag of the same name.
		if e.options.Source != "" {
			tags[SourceField] = e.options.Source
		}

		// SourceType. Overwrite metric tag of the same name.
		if e.options.SourceType != "" {
			tags[SourceTypeField] = e.options.SourceType
		}

		switch rowData := viewRow.Data.(type) {
		case *view.CountData:
			addMetric(viewData.View.Name, float64(rowData.Value), tags)
		case *view.SumData:
			addMetric(viewData.View.Name, rowData.Value, tags)
		case *view.LastValueData:
			addMetric(viewData.View.Name, rowData.Value, tags)
		case *view.DistributionData:
			params := map[string]float64{
				"count": float64(rowData.Count),
				"min":   rowData.Min,
				"max":   rowData.Max,
				"mean":  rowData.Mean,
				"ss":    rowData.SumOfSquaredDev,
			}

			for suffix, value := range params {
				addMetric(
					fmt.Sprintf("%s/%s", viewData.View.Name, suffix),
					value,
					tags)
			}
		}
	}

	if len(metrics) == 0 {
		return
	}

	// Successes and failures will be logged by output.
	_, _ = e.output.PutMetrics(
		context.Background(), &data_receiver.Metrics{
			TaggedMetrics: metrics})
}
