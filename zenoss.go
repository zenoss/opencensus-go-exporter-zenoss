package zenoss

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"regexp"
	"strings"
	"time"

	"go.opencensus.io/stats/view"
	"google.golang.org/api/support/bundler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	zenoss "github.com/zenoss/zenoss-protobufs/go/cloud/data_receiver"
)

const (
	// DefaultAddress is the default Zenoss API endpoint address.
	DefaultAddress = "api.zenoss.io:443"

	// DefaultSourceType is the default source-type identifying this OpenCensus exporter.
	DefaultSourceType = "zenoss/opencensus-go-exporter-zenoss"

	// APIKeyHeader is the gRPC header field containing a Zenoss API key.
	APIKeyHeader = "zenoss-api-key"

	// SourceTypeField is the dimension or metadata field name identifying the source-type.
	SourceTypeField = "source-type"

	// NameField is the model metadata field for an entity's display name.
	NameField = "name"

	// DescriptionField is the optional metric metadata field containing a metric's description.
	DescriptionField = "description"

	// UnitsField is the optional metric metadata field containing a metric's unit of measure.
	UnitsField = "units"
)

type LogLevel int8

const (
	LogLevelDebug   = 1
	LogLevelInfo    = 2
	LogLevelWarning = 3
	LogLevelError   = 4
)

var (
	// Ensure we implement view.Explorer interface.
	_ view.Exporter = (*Exporter)(nil)
)

type Options struct {
	// Address is the Zenoss API endpoint.
	// Default: api.zenoss.io:443
	Address string

	// APIKey is a Zenoss API key.
	// Required.
	APIKey string

	// GlobalDimensions will be added as dimensions to all models and metrics.
	// Optional, but recommended to include at least "source" field.)
	GlobalDimensions map[string]string

	// GlobalMetadataFields will be added as metadata fields to all models and
	// metrics.
	// Optional.
	GlobalMetadataFields map[string]string

	// ModelDimensionTags is a list of OpenCensus tags that will be used as
	// model and metric dimensions.
	// Optional, but recommended.
	ModelDimensionTags []string

	// IgnoredMetricNames is a list of regular expression patterns. Metric
	// names that match any of the patterns will not be exported.
	// Optional.
	IgnoredMetricNames []string

	// BundleDelayThreshold determines the max amount of time the exporter can
	// can wait before sending collected models and metrics to Zenoss.
	// Default: 1 second
	BundleDelayThreshold time.Duration

	// BundleCountThreshold determines how many models and metrics can be
	// buffered before sending to Zenoss.
	// Default: 1000
	BundleCountThreshold int

	// OnViewData is a function that can be used to supply custom logic for
	// creating zenoss.Data from an OpenCensus view.Data.
	// Optional.
	OnViewData func(viewData *view.Data) (data *Data, preventDefault bool)

	// OnViewRow is a function that can be used to supply custom logic for
	// creating zenoss.Data from an OpenCensus view.Row.
	// Optional.
	OnViewRow func(viewData *view.Data, viewRow *view.Row) (data *Data, preventDefault bool)

	// OnData is a function that can be used to supply any custom logic for
	// manipulating zenoss.Data before it is sent to Zenoss.
	// Optional.
	OnData func(data *Data) *Data

	// OnLog is a function that can be used to supply custom logging logic.
	// Optional.
	OnLog func(level LogLevel, fields map[string]interface{}, format string, args ...interface{})
}

type Exporter struct {
	options               Options
	ignoredMetrics        []*regexp.Regexp
	client                zenoss.DataReceiverServiceClient
	modelsBundler         *bundler.Bundler
	metricsBundler        *bundler.Bundler
	modelFreshnessChecker *freshnessChecker
}

func NewExporter(options Options) (*Exporter, error) {
	if options.Address == "" {
		options.Address = DefaultAddress
	}

	// Compile regular expressions for ignored metrics.
	var ignoredMetrics []*regexp.Regexp
	for _, s := range options.IgnoredMetricNames {
		if r, err := regexp.Compile(s); err == nil {
			ignoredMetrics = append(ignoredMetrics, r)
		} else {
			return nil, err
		}
	}

	opt := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	conn, err := grpc.Dial(options.Address, opt)
	if err != nil {
		return nil, err
	}

	e := &Exporter{
		options:               options,
		ignoredMetrics:        ignoredMetrics,
		client:                zenoss.NewDataReceiverServiceClient(conn),
		modelFreshnessChecker: newFreshnessChecker(3600),
	}

	e.modelsBundler = bundler.NewBundler((*zenoss.Model)(nil), func(bundle interface{}) {
		e.putModels(bundle.([]*zenoss.Model))
	})

	e.metricsBundler = bundler.NewBundler((*zenoss.Metric)(nil), func(bundle interface{}) {
		e.putMetrics(bundle.([]*zenoss.Metric))
	})

	if delayThreshold := e.options.BundleDelayThreshold; delayThreshold > 0 {
		e.modelsBundler.DelayThreshold = delayThreshold
		e.metricsBundler.DelayThreshold = delayThreshold
	} else {
		e.modelsBundler.DelayThreshold = 1 * time.Second
		e.metricsBundler.DelayThreshold = 1 * time.Second
	}

	if countThreshold := e.options.BundleCountThreshold; countThreshold > 0 {
		e.modelsBundler.BundleCountThreshold = countThreshold
		e.metricsBundler.BundleCountThreshold = countThreshold
	} else {
		e.modelsBundler.BundleCountThreshold = 1000
		e.metricsBundler.BundleCountThreshold = 1000
	}

	return e, nil
}

// Flush waits for exported data to be sent.
func (e *Exporter) Flush() {
	e.modelsBundler.Flush()
	e.metricsBundler.Flush()
}

// ExportView exports stats to Zenoss.
func (e *Exporter) ExportView(viewData *view.Data) {
	e.log(
		LogLevelDebug,
		logrus.Fields{"viewName": viewData.View.Name},
		"exporting view")

	for _, ignoredMetric := range e.ignoredMetrics {
		if ignoredMetric.FindString(viewData.View.Name) != "" {
			e.log(
				LogLevelDebug,
				logrus.Fields{"viewName": viewData.View.Name},
				"ignoring metric")

			return
		}
	}

	// Use custom OnViewData function if defined.
	if e.options.OnViewData != nil {
		data, preventDefault := e.options.OnViewData(viewData)

		// Allow default processing of view to be prevented.
		if preventDefault {
			e.bundleData(data)
			return
		}
	}

	// Run default onViewData if custom OnViewData didn't preventDefault.
	e.onViewData(viewData)
}

func (e *Exporter) onViewData(viewData *view.Data) {
	for _, viewRow := range viewData.Rows {
		// Use custom OnViewRow function if defined.
		if e.options.OnViewRow != nil {
			data, preventDefault := e.options.OnViewRow(viewData, viewRow)
			e.bundleData(data)

			// Allow default processing of row to be prevented.
			if preventDefault {
				continue
			}
		}

		e.onViewRow(viewData, viewRow)
	}
}

func (e *Exporter) onViewRow(viewData *view.Data, viewRow *view.Row) {
	var nameParts []string
	dimensions := map[string]string{}
	metadataFields := map[string]string{SourceTypeField: DefaultSourceType}

	// Pull dimensions, and name from row's tags.
	for _, modelDimensionTag := range e.options.ModelDimensionTags {
		for _, rowTag := range viewRow.Tags {
			if rowTag.Key.Name() == modelDimensionTag {
				dimensions[modelDimensionTag] = rowTag.Value
				nameParts = append(nameParts, rowTag.Value)
				break
			}
		}
	}

	if len(nameParts) == 0 {
		tags := make(map[string]string, len(viewRow.Tags))
		for _, tag := range viewRow.Tags {
			tags[tag.Key.Name()] = tag.Value
		}

		e.log(
			LogLevelDebug,
			logrus.Fields{"viewName": viewData.View.Name, "tags": tags},
			"ignoring row with no model dimension tags")

		return
	}

	// Any tag that isn't a dimension becomes metadata.
	// This is done separately from the loop above because we want nameParts
	// ordered the same as ModelDimensionTags.
	var tagName string
	for _, rowTag := range viewRow.Tags {
		tagName = rowTag.Key.Name()
		if _, exists := dimensions[tagName]; !exists {
			metadataFields[tagName] = rowTag.Value
		}
	}

	// Add impact fields specific to Kubernetes applications.
	addKubernetesImpacts(metadataFields)

	// Copy global dimensions.
	for k, v := range e.options.GlobalDimensions {
		dimensions[k] = v
	}

	// Copy global metadata fields.
	for k, v := range e.options.GlobalMetadataFields {
		metadataFields[k] = v
	}

	// Row may not have any tags in EntityTagKeys. Nothing to be done for that.
	if len(nameParts) == 0 {
		return
	}

	data := &Data{}

	timestamp := viewData.End.UnixNano() / 1e6

	// Only models need a name field.
	modelMetadataFields := copyMap(metadataFields)
	modelMetadataFields[NameField] = strings.Join(nameParts, "/")

	data.AddModel(&zenoss.Model{
		Timestamp:      timestamp,
		Dimensions:     dimensions,
		MetadataFields: metadataFieldsFromMap(modelMetadataFields),
	})

	addMetric := func(name string, value float64) {
		metricMetadataFields := copyMap(metadataFields)

		description := viewData.View.Description
		if description != "" {
			metricMetadataFields[DescriptionField] = description
		}

		units := viewData.View.Measure.Unit()
		if units != "" {
			metricMetadataFields[UnitsField] = units
		}

		data.AddMetric(&zenoss.Metric{
			Metric:         name,
			Dimensions:     copyMap(dimensions),
			MetadataFields: metadataFieldsFromMap(metricMetadataFields),
			Timestamp:      timestamp,
			Value:          value,
		})
	}

	switch rowData := viewRow.Data.(type) {
	case *view.CountData:
		addMetric(viewData.View.Name, float64(rowData.Value))
	case *view.SumData:
		addMetric(viewData.View.Name, rowData.Value)
	case *view.LastValueData:
		addMetric(viewData.View.Name, rowData.Value)
	case *view.DistributionData:
		params := map[string]float64{
			"count": float64(rowData.Count),
			"min":   rowData.Min,
			"max":   rowData.Max,
			"mean":  rowData.Mean,
			"ss":    rowData.SumOfSquaredDev,
		}

		for suffix, value := range params {
			addMetric(fmt.Sprintf("%s/%s", viewData.View.Name, suffix), value)
		}
	}

	e.bundleData(data)
}

func (e *Exporter) bundleData(data *Data) {
	var err error

	// Allow custom OnData function to manipulate data.
	if e.options.OnData != nil {
		data = e.options.OnData(data)
	}

	for _, model := range data.Models {
		if e.modelFreshnessChecker.isFresh(model) {
			// Avoid sending models that have been sent recently.
			continue
		}

		err = e.modelsBundler.Add(model, 1)
		switch err {
		case nil:
			continue
		case bundler.ErrOverflow:
			e.log(
				LogLevelError,
				logrus.Fields{"error": err},
				"failed to send models: buffer full")
		default:
			e.log(
				LogLevelError,
				logrus.Fields{"error": err},
				"failed to send models")
		}
	}

	for _, metric := range data.Metrics {
		err = e.metricsBundler.Add(metric, 1)
		switch err {
		case nil:
			continue
		case bundler.ErrOverflow:
			e.log(
				LogLevelError,
				logrus.Fields{"error": err},
				"failed to send metrics: buffer full")
		default:
			e.log(
				LogLevelError,
				logrus.Fields{"error": err},
				"failed to send metrics")
		}
	}
}

func (e *Exporter) putModels(models []*zenoss.Model) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, APIKeyHeader, e.options.APIKey)

	modelStatus, err := e.client.PutModels(ctx, &zenoss.Models{
		DetailedResponse: true,
		Models:           models,
	})

	if err != nil {
		e.log(
			LogLevelError,
			logrus.Fields{"error": err, "models": len(models)},
			"unable to send models")
	} else {
		if modelStatus.GetFailed() > 0 {
			e.log(
				LogLevelWarning,
				logrus.Fields{"error": err, "models": len(models), "failed": modelStatus.GetFailed()},
				"failed to send models")
		} else {
			e.log(
				LogLevelDebug,
				logrus.Fields{"models": len(models)},
				"sent models")
		}
	}
}

func (e *Exporter) putMetrics(metrics []*zenoss.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, APIKeyHeader, e.options.APIKey)

	metricStatus, err := e.client.PutMetrics(ctx, &zenoss.Metrics{
		DetailedResponse: true,
		Metrics:          metrics,
	})

	if err != nil {
		e.log(
			LogLevelError,
			logrus.Fields{"error": err, "metrics": len(metrics)},
			"unable to send metrics")
	} else {
		if metricStatus.GetFailed() > 0 {
			e.log(
				LogLevelWarning,
				logrus.Fields{"error": err, "metrics": len(metrics), "failed": metricStatus.GetFailed()},
				"failed to send metrics")
		} else {
			e.log(
				LogLevelDebug,
				logrus.Fields{"metrics": len(metrics)},
				"sent metrics")
		}
	}
}

func (e *Exporter) log(level LogLevel, fields map[string]interface{}, format string, args ...interface{}) {
	if e.options.OnLog != nil {
		// Use customer log function if available.
		e.options.OnLog(level, fields, format, args...)
	} else if level > LogLevelInfo {
		// Otherwise log only warnings and errors.
		log.Printf(fmt.Sprintf("%s fields=%+v", format, fields), args...)
	}
}
