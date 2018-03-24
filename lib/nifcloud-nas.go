package mpnifcloudnas


import (
	"errors"
	"flag"
	"log"
	"strings"
	"sync"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

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

func getLastPoint(client *NasClient, identifier, metricName string) (float64, error) {
	now := time.Now().In(time.UTC)
	const layout = "2006-01-02 15:04:05"
	params := make(map[string]string)
	params["Dimensions.member.1.Name"] = "NASInstanceIdentifier"
	params["Dimensions.member.1.Value"] = identifier
	params["EndTime"] = now.Format(layout)
	params["StartTime"] = now.Add(time.Duration(180) * time.Second * -1).Format(layout) // 3 min (to fetch at least 1 data-point)
	params["MetricName"] = metricName

	response, err := client.GetMetricStatistics(params)
	if err != nil {
		return 0, err
	}

	datapoints := response.GetMetricStatisticsResult.Datapoints
	if len(datapoints) == 0 {
		return 0, errors.New("fetched no datapoints")
	}
	members := datapoints[0].Member
	if len(members) == 0 {
		return 0, errors.New("fetched no members")
	}

	latest := new(time.Time)
	var latestVal float64
	for _, m := range members {
		if m.Timestamp.Before(*latest) {
			continue
		}

		latest = &m.Timestamp
		latestVal = float64(m.Sum) / float64(m.SampleCount)
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
	client := NewNasClient(p.Region, p.AccessKeyID, p.SecretAccessKey)
	stat := make(map[string]float64)
	var wg sync.WaitGroup
	for _, met := range p.nasMetrics() {
		wg.Add(1)
		go func(met string) {
			defer wg.Done()
			v, err := getLastPoint(client, p.Identifier, met)
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