package zenoss

import (
	"crypto/tls"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"sync"

	"go.opencensus.io/stats/view"

	zenoss "github.com/zenoss/zenoss-protobufs/go/cloud/data_receiver"
)


var (
	// Ensure we implement view.Explorer interface.
	_ view.Exporter  = (*Exporter)(nil)
)

type Options struct {
	Address string
	APIKey string
	Source string
	Dimensions map[string]string
	MetadataFields map[string]string
	OnError func(err error)
}

func (options *Options) onError(err error) {
	if options.OnError != nil {
		options.OnError(err)
	} else {
		log.Printf("Failed to export to Zenoss: %v\n", err)
	}
}

type Exporter struct {
	options Options
	client zenoss.DataReceiverServiceClient
	mu sync.Mutex
	viewData map[string]*view.Data
}

func NewExporter(options Options) (*Exporter, error) {
	opt := grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	conn, err := grpc.Dial(options.Address, opt)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		options: options,
		client:  zenoss.NewDataReceiverServiceClient(conn),
		viewData: make(map[string]*view.Data),
	}, nil
}

func (e *Exporter) ExportView(viewData *view.Data) {
	e.options.onError(fmt.Errorf("zenoss stats export not implemented"))
}
