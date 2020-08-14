package v1
// from https://github.com/reachlin/sensu_exporter, only change from main function to V1 for package v1

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
  	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var (
	timeout       = flag.Duration(
		"timeout", 
		20, 
		"Timeout in seconds for the API request",
	)
	listenAddress = flag.String(
		// exporter port list:
		// https://github.com/prometheus/prometheus/wiki/Default-port-allocations
		"listen", ":9251",
		"Address to listen on for serving Prometheus Metrics. Only need for sensu v1",
	)
	sensuAPI = flag.String(
		"api", "http://sensu.dev.mycompany.com:4567",
		"Address to Sensu API. Only need for sensu v1",
	)
)

type SensuCheckResult struct {
	Client string
	Check  SensuCheck
}

type SensuCheck struct {
	Name        string
	Duration    float64
	Executed    int64
	Subscribers []string
	Output      string
	Status      int
	Issued      int64
	Interval    int
}

// BEGIN: Class SensuCollector
type SensuCollector struct {
	apiUrl      string
	mutex       sync.RWMutex
	cli         *http.Client
	CheckStatus *prometheus.Desc
}

func (c *SensuCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.CheckStatus
}

func (c *SensuCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock() // To protect metrics from concurrent collects.
	defer c.mutex.Unlock()

	results := c.getCheckResults()
	for i, result := range results {
		log.Debugln("...", fmt.Sprintf("%d, %v, %v", i, result.Check.Name, result.Check.Status))
		// in Sensu, 0 means OK
		// in Prometheus, 1 means OK
		status := 0.0
		if result.Check.Status == 0 {
			status = 1.0
		} else {
			status = 0.0
		}
		ch <- prometheus.MustNewConstMetric(
			c.CheckStatus,
			prometheus.GaugeValue,
			status,
			result.Client,
			result.Check.Name,
		)
	}
}

func (c *SensuCollector) getCheckResults() []SensuCheckResult {
	log.Debugln("Sensu API URL", c.apiUrl)
	results := []SensuCheckResult{}
	err := c.GetJson(c.apiUrl+"/results", &results)
	if err != nil {
		log.Errorln("Query Sensu failed.", fmt.Sprintf("%v", err))
	}
	return results
}

func (c *SensuCollector) GetJson(url string, obj interface{}) error {
	resp, err := c.cli.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(obj)
}

// END: Class SensuCollector

func NewSensuCollector(url string, cli *http.Client) *SensuCollector {
	return &SensuCollector{
		cli:    cli,
		apiUrl: url,
		CheckStatus: prometheus.NewDesc(
			"sensu_check_status",
			"Sensu Check Status(1:Up, 0:Down)",
			[]string{"client", "check_name"},
			nil,
		),
	}
}

// V1 is the entry function for sensu V1
func V1() {
	flag.Parse()

	collector := NewSensuCollector(*sensuAPI, &http.Client{
		Timeout: *timeout,
	})
	fmt.Println(collector.cli.Timeout)
	prometheus.MustRegister(collector)
	metricPath := "/metrics"
	http.Handle(metricPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(metricPath))
	})
	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
