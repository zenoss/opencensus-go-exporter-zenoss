package zenoss

import (
	zenoss "github.com/zenoss/zenoss-protobufs/go/cloud/data_receiver"
)

type Data struct {
	Models  []*zenoss.Model
	Metrics []*zenoss.Metric
}

func (d *Data) AddModel(model *zenoss.Model) {
	d.Models = append(d.Models, model)
}

func (d *Data) AddMetric(metric *zenoss.Metric) {
	d.Metrics = append(d.Metrics, metric)
}
