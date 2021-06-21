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
	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v2"
)

var (
	dashboardFile  = kingpin.Flag("file.dashboard", "Grafana dashboard JSON file to convert.").Short('f').Required().ExistingFile()
	manifestFile   = kingpin.Flag("file.output", "Output file for the dashboard configmap.").Short('o').Default("").String()
	compact        = kingpin.Flag("file.compact", "Output file with compact JSON embedded in ConfigMap.").Short('c').Default("false").Bool()
	dashboardName  = kingpin.Flag("dashboard.name", "Dashboard configmap name. (Default: dashboard file basename)").Short('n').Default("").String()
	k8sAnnotations = kingpin.Flag("k8s.annotations", "Add an annotation to add the dashboard configmap (key=value)").Short('a').StringMap()
	k8sNamespace   = kingpin.Flag("k8s.namespace", "kubernetes namespace for the configmap.").Short('N').Default("monitoring").String()
)

type configMapMetadataLabels struct {
	GrafanaDashboard string `yaml:"grafana_dashboard"`
}

type configMapMetadata struct {
	Name        string                  `yaml:"name"`
	Namespace   string                  `yaml:"namespace"`
	Labels      configMapMetadataLabels `yaml:"labels"`
	Annotations map[string]string       `yaml:"annotations,omitempty"`
}

type grafanaConfigMap struct {
	ApiVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   configMapMetadata `yaml:"metadata"`
	Data       map[string]string `yaml:"data,omitempty"`
}

func readDashboardJson(file string) (*models.Dashboard, error) {
	fh, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("Error: %s could not be opened (%s)", file, err)
	}
	dbj, err := simplejson.NewFromReader(fh)
	if err != nil {
		return nil, fmt.Errorf("Error: %s contents could not be converted to simplejson (%s)", file, err)
	}
	dbo := models.NewDashboardFromJson(dbj)
	return dbo, nil
}

// ProcessDashboardFile(dashboardFile, manifestFile, namespace, name, compact, annotaitons) error
// Given a dashboard json file, will generate a k8s ConfigMap and write it to the manifestFile location
func ProcessDashboardFile(dbf, mff, ns, n string, c bool, as *map[string]string) (err error) {
	if !strings.HasSuffix(dbf, ".json") {
		return fmt.Errorf("%s is not a json file", dbf)
	}

	bdf := filepath.Base(dbf)
	bdfns := strings.TrimSuffix(bdf, ".json")
	if mff == "" {
		mff = fmt.Sprintf("%s.yaml", bdfns)
	}
	if n == "" {
		n = strings.Replace(bdfns, "_", "-", -1)
	}

	db, err := readDashboardJson(dbf)
	if err != nil {
		return err
	}
	d := db.Data
	_, err = d.Encode()
	if err != nil {
		return err
	}

	var dp []byte
	if c {
		dp, err = d.Encode()
	} else {
		dp, err = d.EncodePretty()
	}
	if err != nil {
		return err
	}

	cm := grafanaConfigMap{
		ApiVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: configMapMetadata{
			Name: n,
			Labels: configMapMetadataLabels{
				GrafanaDashboard: "1",
			},
			Annotations: *as,
		},
		Data: map[string]string{bdf: fmt.Sprintln(string(dp))},
	}
	if ns != "" {
		cm.Metadata.Namespace = ns
	}
	md, err := yaml.Marshal(&cm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(mff, md, 0666)
	return
}

func main() {
	log.Logger = log.With().Caller().Logger()
	kingpin.Parse()
	err := ProcessDashboardFile(*dashboardFile, *manifestFile, *k8sNamespace, *dashboardName, *compact, k8sAnnotations)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
