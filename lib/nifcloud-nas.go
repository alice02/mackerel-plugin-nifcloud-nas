package mpnifcloudnas

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aokumasan/nifcloud-sdk-go-v2/nifcloud"
	"github.com/aokumasan/nifcloud-sdk-go-v2/service/nas"
	"github.com/aws/aws-sdk-go-v2/private/protocol/query/queryutil"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

const timestampLayout = "2006-01-02 15:04:05"

// NASPlugin mackerel plugin for NIFCLOUD NAS
type NASPlugin struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Identifier      string
	Engine          string
	Prefix          string
	LabelPrefix     string
}

func getLastPoint(client *nas.Client, dimension nas.RequestDimensionsStruct, metricName string) (float64, error) {
	now := time.Now().In(time.UTC)

	request := client.GetMetricStatisticsRequest(&nas.GetMetricStatisticsInput{
		Dimensions: []nas.RequestDimensionsStruct{dimension},
		MetricName: nifcloud.String(metricName),
	})

	// This is workaround for NAS request parameter datetime format
	if err := request.Build(); err != nil {
		return 0, fmt.Errorf("failed building request: %v", err)
	}
	body := url.Values{
		"Action":  {request.Operation.Name},
		"Version": {request.Metadata.APIVersion},
	}
	if err := queryutil.Parse(body, request.Params, false); err != nil {
		return 0, fmt.Errorf("failed encoding request: %v", err)
	}
	body.Set("StartTime", now.Add(time.Duration(180)*time.Second*-1).Format(timestampLayout)) // 3 min (to fetch at least 1 data-point)
	body.Set("EndTime", now.Format(timestampLayout))
	request.SetBufferBody([]byte(body.Encode()))

	response, err := request.Send(context.Background())
	if err != nil {
		return 0, err
	}

	datapoints := response.Datapoints
	if len(datapoints) == 0 {
		return 0, errors.New("fetched no datapoints")
	}

	latest := new(time.Time)
	var latestVal float64
	for _, dp := range datapoints {
		timestamp, err := time.Parse(time.RFC3339, nifcloud.StringValue(dp.Timestamp))
		if err != nil {
			return 0, fmt.Errorf("could not parse timestamp %q: %v", nifcloud.StringValue(dp.Timestamp), err)
		}

		if timestamp.Before(*latest) {
			continue
		}

		latest = &timestamp
		latestVal, err = strconv.ParseFloat(nifcloud.StringValue(dp.Sum), 64)
		if err != nil {
			return 0, fmt.Errorf("could not parse sum %q: %v", nifcloud.StringValue(dp.Sum), err)
		}
	}
	return latestVal, nil
}

func (p NASPlugin) nasMetrics() (metrics []string) {
	for _, v := range p.GraphDefinition() {
		for _, vv := range v.Metrics {
			metrics = append(metrics, vv.Name)
		}
	}
	return
}

// FetchMetrics interface for mackerel-plugin
func (p NASPlugin) FetchMetrics() (map[string]float64, error) {
	nasClient := nas.New(nifcloud.NewConfig(p.AccessKeyID, p.SecretAccessKey, p.Region))
	perInstance := nas.RequestDimensionsStruct{
		Name:  nifcloud.String("NASInstanceIdentifier"),
		Value: nifcloud.String(p.Identifier),
	}
	stat := make(map[string]float64)
	var wg sync.WaitGroup
	for _, met := range p.nasMetrics() {
		wg.Add(1)
		go func(met string) {
			defer wg.Done()
			v, err := getLastPoint(nasClient, perInstance, met)
			if err == nil {
				stat[met] = v
			} else {
				log.Printf("%s: %s", met, err)
			}
		}(met)
	}
	wg.Wait()

	return stat, nil
}

// GraphDefinition interface for mackerel plugin
func (p NASPlugin) GraphDefinition() map[string]mp.Graphs {
	return map[string]mp.Graphs{
		p.Prefix + ".FreeStorageSpace": {
			Label: p.LabelPrefix + " Free Storage Space",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "FreeStorageSpace", Label: "FreeStorageSpace"},
			},
		},
		p.Prefix + ".UsedStorageSpace": {
			Label: p.LabelPrefix + " Used Storage Space",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "UsedStorageSpace", Label: "UsedStorageSpace"},
			},
		},
		p.Prefix + ".ActiveConnections": {
			Label: p.LabelPrefix + " Active Connections",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: "ActiveConnections", Label: "ActiveConnections"},
			},
		},
		p.Prefix + ".IOPS": {
			Label: p.LabelPrefix + " IOPS",
			Unit:  "iops",
			Metrics: []mp.Metrics{
				{Name: "ReadIOPS", Label: "Read"},
				{Name: "WriteIOPS", Label: "Write"},
			},
		},
		p.Prefix + ".Throughput": {
			Label: p.LabelPrefix + " Throughput",
			Unit:  "bytes/sec",
			Metrics: []mp.Metrics{
				{Name: "ReadThroughput", Label: "Read"},
				{Name: "WriteThroughput", Label: "Write"},
			},
		},
		p.Prefix + ".GlobalTraffic": {
			Label: p.LabelPrefix + " Global Traffic",
			Unit:  "bytes/sec",
			Metrics: []mp.Metrics{
				{Name: "GlobalReadTraffic", Label: "Read"},
				{Name: "GlobalWriteTraffic", Label: "Write"},
			},
		},
		p.Prefix + ".PrivateTraffic": {
			Label: p.LabelPrefix + " Private Traffic",
			Unit:  "bytes/sec",
			Metrics: []mp.Metrics{
				{Name: "PrivateReadTraffic", Label: "Read"},
				{Name: "PrivateWriteTraffic", Label: "Write"},
			},
		},
	}
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p NASPlugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		p.Prefix = "nas"
	}
	return p.Prefix
}

// Do the plugin
func Do() {
	optRegion := flag.String("region", "", "Region")
	optAccessKeyID := flag.String("access-key-id", "", "Access Key ID")
	optSecretAccessKey := flag.String("secret-access-key", "", "Secret Access Key")
	optIdentifier := flag.String("identifier", "", "NAS Instance Identifier")
	optPrefix := flag.String("metric-key-prefix", "nas", "Metric key prefix")
	optLabelPrefix := flag.String("metric-label-prefix", "", "Metric Label prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	nas := NASPlugin{
		Prefix: *optPrefix,
	}
	if *optLabelPrefix == "" {
		if *optPrefix == "nas" {
			nas.LabelPrefix = "NAS"
		} else {
			nas.LabelPrefix = strings.Title(*optPrefix)
		}
	} else {
		nas.LabelPrefix = *optLabelPrefix
	}

	nas.Region = *optRegion
	nas.Identifier = *optIdentifier
	nas.AccessKeyID = *optAccessKeyID
	nas.SecretAccessKey = *optSecretAccessKey

	helper := mp.NewMackerelPlugin(nas)
	helper.Tempfile = *optTempfile

	helper.Run()
}
