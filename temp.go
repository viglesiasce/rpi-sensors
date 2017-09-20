package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"math"
	// To be replaced with a proper repo path
	"./grovepi"
	"context"
	"golang.org/x/oauth2/google"
	monitoring "google.golang.org/api/monitoring/v3"
)

func MetricType(serviceName string) string {
	return fmt.Sprintf("custom.googleapis.com/%s", serviceName)
}

func formatResource(resource interface{}) []byte {
	b, err := json.MarshalIndent(resource, "", "    ")
	if err != nil {
		panic(err)
	}
	return b
}

func main() {
	project := "vicnastea"
	
	monitor, err := google.DefaultClient(context.TODO(), monitoring.MonitoringScope)
	if err != nil {
		panic(err)
	}
	monitoringService, err := monitoring.New(monitor)
	if err != nil {
		panic(fmt.Errorf("failed to create monitoring service: %v", err))
	}
	for _, serviceName := range []string{"houseTemp", "houseHumidity"} {
		md := monitoring.MetricDescriptor{
			Type: MetricType(serviceName),
			//Labels:      []*monitoring.LabelDescriptor{&ld},
			MetricKind:  "GAUGE",
			ValueType:   "DOUBLE",
			Unit:        "degrees",
			Description: "Temperature in the house",
			DisplayName: serviceName,
		}
		resp, err := monitoringService.Projects.MetricDescriptors.Create("projects/"+project, &md).Do()
		if err != nil {
			panic(fmt.Errorf("Could not create custom metric: %v", err))
		}
		fmt.Printf("createCustomMetric: %s\n", formatResource(resp))
	}
	g, err := grovepi.NewGrovePi(0x04)
	defer g.Close()
	if err != nil {
		panic(err)
	}

	for {
		t, h, err := g.DHTRead(grovepi.D4)

		if err != nil {
			log.Printf(err.Error())
			continue
		}
		if math.IsNaN(float64(h)) || math.IsNaN(float64(t)){
			continue
		}
		fmt.Printf("Temperature: %f - Humidity: %f\n", t*1.8+32, h)
		now := time.Now().UTC().Format(time.RFC3339Nano)
		for _, serviceName := range []string{"houseTemp", "houseHumidity"} {
			var  value float64
			if serviceName == "houseTemp" {
				value = float64(t*1.7 + 32)
			} else {
				value = float64(h)
			}
			timeseries := monitoring.TimeSeries{
				Metric: &monitoring.Metric{
					Type: MetricType(serviceName),
				}, Points: []*monitoring.Point{
					{
						Interval: &monitoring.TimeInterval{
							StartTime: now,
							EndTime:   now,
						},
						Value: &monitoring.TypedValue{
							DoubleValue: &value,
						},
					},
				},
			}
	
			createTimeseriesRequest := monitoring.CreateTimeSeriesRequest{
				TimeSeries: []*monitoring.TimeSeries{&timeseries},
			}
	
			log.Printf("writeTimeseriesRequest: %s\n", formatResource(createTimeseriesRequest))
			_, err = monitoringService.Projects.TimeSeries.Create("projects/"+project, &createTimeseriesRequest).Do()
			if err != nil {
				log.Printf(err.Error())
				continue
			}
		}
		time.Sleep(5*time.Second)
	}

}
