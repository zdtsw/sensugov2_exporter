package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
	"os"
	"github.com/magiconair/properties"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

type sensuEvent struct {
	Check  sensuCheck
	Entity sensuEntity
}

type sensuCheck struct {
	State       string
	Status      int
	Output      string
	Issued      int64
	Occurrences int
	Duration    float64
	Executed    int64
	Interval    int
	Subscribers []string
	Metadata    checkMetadata
}

type sensuEntity struct {
	Class  string
	System sensuSystem
}

type sensuSystem struct {
	Hostname string
}

type checkMetadata struct {
	Name      string
	Namespace string
}

// SensuCollector Type Declariaion begin
type SensuCollector struct {
	apiURL      string
	apiAuthKey  string
	namespace   string
	mutex       sync.RWMutex
	cli         http.Client
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
		log.Debugln("...", fmt.Sprintf("%d, %v, %v", i, result.Check.Metadata.Name, result.Check.Status))
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
			result.Entity.System.Hostname,
			result.Check.Metadata.Name,
		)
	}
}


func (c *SensuCollector) getCheckResults() []sensuEvent {
	log.Debugln("Sensu API URL", c.apiURL)
	results := []sensuEvent{}
	err := c.getJSON(&results)
	if err != nil {
		log.Errorln("Query SensuGo failed.", fmt.Sprintf("%v", err))
	}

	return results
}

func (c *SensuCollector) getJSON(obj interface{}) error {
	req, err := http.NewRequest("GET", c.apiURL+"/api/core/v2/namespaces/"+c.namespace+"/events", nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Authorization", "Key "+c.apiAuthKey)

	resp, err := c.cli.Do(req)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}

	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(obj)
}

// SensuCollector Type Declariaion End

func newSensuCollector(url string, namespace string, authKey string, clientTimeout int) *SensuCollector {
	return &SensuCollector{
		cli: http.Client{
			Timeout: time.Second * time.Duration(clientTimeout),
		},
		apiURL:     url,
		apiAuthKey: authKey,
		namespace:  namespace,
		CheckStatus: prometheus.NewDesc(
			"sensu_check_status",
			"Sensu Check Status(1:Up, 0:Down)",
			[]string{"client", "check_name"},
			nil,
		),
	}
}

// V2 is the entry function for sensu-go V2
func V2() {
	pwd, _ := os.Getwd()
	props := properties.MustLoadFile(pwd+"/sensugo_exporter.properties", properties.UTF8)

	sensuAPIUrl := props.GetString("sensuAPIUrl", "")
	sensuNamespace := props.GetString("sensuNamespace", "default")
	sensuAPIAuthKey := props.GetString("sensuAPIAuthKey", "")
	listenAddress := props.GetString("listenAddress", ":9251")
	clientTimeout := props.GetInt("clientTimeout", 20)

	if sensuAPIUrl == "" {
		log.Fatal("Failed to start: missing Sensu API Url")
	}

	if sensuAPIAuthKey == "" {
		log.Fatal("Failed to start: missing Sensu API Auth Key")
	}

	collector := newSensuCollector(sensuAPIUrl, sensuNamespace, sensuAPIAuthKey, clientTimeout)

	fmt.Println("Sensu API Timeout: " + collector.cli.Timeout.String())
	prometheus.MustRegister(collector)
	metricPath := "/metrics"
	http.Handle(metricPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(metricPath))
	})
	log.Infoln("Listening on", listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}
