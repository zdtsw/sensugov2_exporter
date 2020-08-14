package main

import (
	"flag"
	"fmt"
	"github.com/zdtsw/sensugov2_exporter/v1"
	"github.com/zdtsw/sensugov2_exporter/v2"
)

func main() {
	v := flag.String("version", "", "sensu version")

	flag.Parse()
	switch *v {
	case "v2":
		fmt.Println("Start service on V2")
		v2.V2()
	case "v1":
		fmt.Println("Start service on V1")
		v1.V1()
	default:
		fmt.Println("Do not know Sensu version, exit service")
	}
}
