package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/models"
	yaml "gopkg.in/yaml.v2"
)

var (
	dashboardFile = kingpin.Flag("file.dashboard", "Grafana dashboard JSON file to convert.").Short('f').Required().ExistingFile()
	manifestFile  = kingpin.Flag("file.output", "Output file for the dashboard configmap.").Short('o').Default("").String()
	dashboardName = kingpin.Flag("dashboard.name", "Dashboard configmap name. (Default: dashboard file basename)").Short('n').Default("").String()
	k8sNamespace  = kingpin.Flag("k8s.namespace", "kubernetes namespace for the configmap.").Short('N').Default("monitoring").String()
)

type configMapMetadataLabels struct {
	GrafanaDashboard string `yaml:"grafana_dashboard"`
}

type configMapMetadata struct {
	Name      string                  `yaml:"name"`
	Namespace string                  `yaml:"namespace"`
	Labels    configMapMetadataLabels `yaml:"labels"`
}

type grafanaConfigMap struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   configMapMetadata `yaml:"metadata"`
	Data       map[string]string `yaml:"data,omitempty"`
}

func readDashboardJson(file string) *models.Dashboard {
	fh, err := os.Open(file)
	if err != nil {
		panic(fmt.Sprintf("Error: %s could not be opened (%s)", file, err))
	}
	dbj, err := simplejson.NewFromReader(fh)
	if err != nil {
		panic(fmt.Sprintf("Error: %s contents could not be converted to simplejson (%s)", file, err))
	}
	dbo := models.NewDashboardFromJson(dbj)
	return dbo
}

func main() {
	kingpin.Parse()
	if !strings.HasSuffix(*dashboardFile, ".json") {
		panic(fmt.Sprintf("%s is not a file...exiting", *dashboardFile))
	}
	//dfns := strings.TrimSuffix(*dashboardFile, ".json")
	bdf := filepath.Base(*dashboardFile)
	bdfns := strings.TrimSuffix(bdf, ".json")
	if *manifestFile == "" {
		*manifestFile = fmt.Sprintf("%s.yaml", bdfns)
	}
	if *dashboardName == "" {
		*dashboardName = strings.Replace(bdfns, "_", "-", -1)
	}

	db := readDashboardJson(*dashboardFile)
	d := db.Data
	dd, err := d.Encode()
	if err != nil {
		panic(err)
	}
	cm := grafanaConfigMap{
		ApiVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: configMapMetadata{
			Name:      bdfns,
			Namespace: *k8sNamespace,
			Labels: configMapMetadataLabels{
				GrafanaDashboard: "1",
			},
		},
		Data: map[string]string{bdf: string(dd)},
	}
	md, err := yaml.Marshal(&cm)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(*manifestFile, md, 0666)
}
