// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consul

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/prometheus/util/testutil"
)

var promPath = filepath.Join("..", "..", "cmd", "prometheus")
var promConfig = filepath.Join("..", "..", "documentation", "examples", "consul-sd.yaml")
var promData = filepath.Join(os.TempDir(), "data")

type ConsulResponse struct {
	Status string `json:"status"`
	Data   struct {
		ActiveTargets []struct {
			DiscoveredLabels struct {
				Address                                string `json:"__address__"`
				MetaConsulAddress                      string `json:"__meta_consul_address"`
				MetaConsulDc                           string `json:"__meta_consul_dc"`
				MetaConsulMetadataConsulNetworkSegment string `json:"__meta_consul_metadata_consul_network_segment"`
				MetaConsulNode                         string `json:"__meta_consul_node"`
				MetaConsulService                      string `json:"__meta_consul_service"`
				MetaConsulServiceAddress               string `json:"__meta_consul_service_address"`
				MetaConsulServiceID                    string `json:"__meta_consul_service_id"`
				MetaConsulServicePort                  string `json:"__meta_consul_service_port"`
				MetaConsulTaggedAddressLan             string `json:"__meta_consul_tagged_address_lan"`
				MetaConsulTaggedAddressWan             string `json:"__meta_consul_tagged_address_wan"`
				MetaConsulTags                         string `json:"__meta_consul_tags"`
				MetricsPath                            string `json:"__metrics_path__"`
				Scheme                                 string `json:"__scheme__"`
				Job                                    string `json:"job"`
			} `json:"discoveredLabels"`
			Labels struct {
				Instance string `json:"instance"`
				Job      string `json:"job"`
			} `json:"labels"`
			ScrapeURL  string    `json:"scrapeUrl"`
			LastError  string    `json:"lastError"`
			LastScrape time.Time `json:"lastScrape"`
			Health     string    `json:"health"`
		} `json:"activeTargets"`
		DroppedTargets []interface{} `json:"droppedTargets"`
	} `json:"data"`
}

func TestConsulDiscovery(t *testing.T) {

	// Compile the prometheus binary in the testdata folder
	buildPrometheus(t)

	//Run the consul server with the binary
	consul := runConsulServer(t)

	//Start the prometheus server with consul SD
	prom := startPrometheus(t)
	consul.Run()

	err := prom.Run()
	// Test the target services length
	time.Sleep(10000 * time.Millisecond)
	resp, err := http.Get("http://localhost:9090/api/v1/targets")
	if err != nil {
		testutil.NotOk(t, err, "")
	} else {
		defer resp.Body.Close()
		consulResponse := ConsulResponse{}
		json.NewDecoder(resp.Body).Decode(consulResponse)
		fmt.Println(len(consulResponse.Status))
		if len(consulResponse.Data.ActiveTargets) == 2 {
			testutil.Ok(t, err)
		}
	}
	// Kill prometheus and consul services
	killServices(prom, consul)
	os.RemoveAll("testdata/prometheues")

}

func buildPrometheus(t *testing.T) {
	os.Setenv("no_proxy", "localhost,127.0.0.1,0.0.0.0,:")

	currPath, err := os.Getwd()
	if err != nil {
		t.Errorf("execution error: %v", err)
		return
	}
	testPath := filepath.Join(currPath, "testdata")
	promBinaryPath := filepath.Join(testPath, "prometheus")

	build := exec.Command("go", "build", "-o", promBinaryPath, promPath)
	output, err := build.CombinedOutput()
	if err != nil {
		fmt.Printf("compilation error :%s \n", output)
		os.Exit(1)
	}
}

func runConsulServer(t *testing.T) *exec.Cmd {
	currPath, err := os.Getwd()
	if err != nil {
		fmt.Printf("can't get current dir :%s \n", err)
		os.Exit(1)
	}
	testPath := filepath.Join(currPath, "testdata")
	consulBinaryPath := filepath.Join(testPath, "consul")
	consul := exec.Command(consulBinaryPath, "agent", "-server", "-bind=127.0.0.1", "-bootstrap-expect=1", "-data-dir=testdata/consul_data", "-config-dir=testdata/consul.d")
	err = consul.Start()
	if err != nil {
		t.Errorf("execution error: %v", err.Error())
		return nil
	}

	return consul
}

func startPrometheus(t *testing.T) *exec.Cmd {
	currPath, err := os.Getwd()
	if err != nil {
		fmt.Printf("can't get current dir :%s \n", err)
		os.Exit(1)
	}
	testPath := filepath.Join(currPath, "testdata")
	promBinaryPath := filepath.Join(testPath, "prometheus")
	fmt.Println(promBinaryPath)
	fmt.Println(promConfig)
	prom := exec.Command(promBinaryPath, "--config.file="+promConfig)
	err = prom.Start()
	if err != nil {
		t.Errorf("execution error: %v", err)
		return nil
	}
	return prom
}

func killServices(prom, consul *exec.Cmd) {
	//consul.Process.Kill()
	prom.Process.Kill()
}
