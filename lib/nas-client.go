package mpnifcloudnas

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
)

const (
	apiVersion = "2016-02-24"
	service    = "nas"
)

// NasClient ...
type NasClient struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

// NiftyGetMetricsStatisticsResponse ...
type GetMetricsStatisticsResponse struct {
	GetMetricStatisticsResult GetMetricsStatisticsResult `xml:"NiftyGetMetricStatisticsResult"`
	ResponseMetadata               ResponseMetadata                `xml:"ResponseMetadata"`
}

// NiftyGetMetricsStatisticsResult ...
type GetMetricsStatisticsResult struct {
	Datapoints []DataPoints `xml:"Datapoints"`
	Label      string       `xml:"Label"`
}

// ResponseMetadata ...
type ResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// DataPoints ...
type DataPoints struct {
	Member []Member `xml:"member"`
}

// Member ...
type Member struct {
	NiftyTargetName string
	Timestamp       time.Time
	Sum             float64
	SampleCount     int
}

func getEndpointFromRegion(region string) (string, error) {
	var endpoints = map[string]string{
		"east-1": "https://nas.jp-east-1.api.cloud.nifty.com/",
		"east-2": "https://nas.jp-east-2.api.cloud.nifty.com/",
		"east-3": "https://nas.jp-east-3.api.cloud.nifty.com/",
		"east-4": "https://nas.jp-east-4.api.cloud.nifty.com/",
		"west-1": "https://nas.jp-west-1.api.cloud.nifty.com/",
	}
	v, ok := endpoints[region]
	if !ok {
		return "", fmt.Errorf("An invalid region was specified")
	}
	return v, nil
}

// NewNasClient ...
func NewNasClient(region, accessKeyID, secretAccessKey string) *NasClient {
	fmt.Println(region, accessKeyID, secretAccessKey)
	return &NasClient{
		Region:          region,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}
}

// Request ...
func (c *NasClient) Request(action string, params map[string]string) ([]byte, error) {
	values := url.Values{}
	values.Set("Action", action)
	for k, v := range params {
		values.Set(k, v)
	}
	body := strings.NewReader(values.Encode())
	endpoint, err := getEndpointFromRegion(c.Region)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	signer := v4.NewSigner(
		credentials.NewStaticCredentials(c.AccessKeyID, c.SecretAccessKey, ""),
	)
	_, err = signer.Sign(req, body, service, c.Region, time.Now())
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// GetMetricStatistics ...
func (c *NasClient) GetMetricStatistics(params map[string]string) (response GetMetricsStatisticsResponse, err error) {
	body, err := c.Request("GetMetricStatistics", params)
	if err != nil {
		return
	}
	fmt.Println(string(body))
	err = xml.Unmarshal(body, &response)
	if err != nil {
		return
	}
	return
}