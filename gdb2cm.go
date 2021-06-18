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
	dashboardsDirGlob = kingpin.Flag("dir.dashboard", "Glob of directories with Grafana dashboard JSON files to convert.").Short('d').String()
	dashboardFile     = kingpin.Flag("file.dashboard", "Grafana dashboard JSON file to convert.").Short('f').ExistingFile()
	manifestsDir      = kingpin.Flag("dir.output", "Output directory for the dashboard configmaps.").Short('m').Default("").ExistingDir()
	cleanManifestsDir = kingpin.Flag("dir.clean", "Clean files in the manifests output directory with this suffix.").Default("").String()
	manifestFile      = kingpin.Flag("file.output", "Output file for the dashboard configmap.").Short('o').Default("").String()
	parentDirAsTeam   = kingpin.Flag("dashboard.team", "If true, use the parent directory name as the team name for ConfigMap names.  Only used if dashboardsDir/manifestsDir set.").Default("false").Bool()
	compact           = kingpin.Flag("file.compact", "Output file with compact JSON embedded in ConfigMap.").Short('c').Default("false").Bool()
	dashboardName     = kingpin.Flag("dashboard.name", "Dashboard configmap name. (Default: dashboard file basename)").Short('n').Default("").String()
	k8sAnnotations    = kingpin.Flag("k8s.annotations", "Add an annotation to add the dashboard configmap (key=value)").Short('a').StringMap()
	k8sNamespace      = kingpin.Flag("k8s.namespace", "kubernetes namespace for the configmap.").Short('N').Default("monitoring").String()
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

func readDashboardJson(file string) *models.Dashboard {
	fh, err := os.Open(file)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Error: %s could not be opened (%s)", file, err))
	}
	dbj, err := simplejson.NewFromReader(fh)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Error: %s contents could not be converted to simplejson (%s)", file, err))
	}
	dbo := models.NewDashboardFromJson(dbj)
	return dbo
}

func processDir(dbDirGlob, mDir string) {
	dirs, err := filepath.Glob(dbDirGlob)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	for _, dir := range dirs {
		pre := ""
		if *parentDirAsTeam {
			pre = fmt.Sprintf("%s-", filepath.Base(filepath.Dir(dir)))
		}
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatal().Msg(err.Error())
			}
			if filepath.Ext(path) != ".json" {
				return nil
			}
			fnns := strings.TrimSuffix(filepath.Base(path), ".json")
			name := fmt.Sprintf("%s%s", pre, fnns)
			mp := filepath.Join(mDir, fmt.Sprintf("%s.db.configmap.yaml", name))
			processFile(path, mp, name)
			return nil
		})
	}
}

func processFile(d, m, n string) {
	if !strings.HasSuffix(d, ".json") {
		log.Fatal().Msg(fmt.Sprintf("%s is not a file...exiting", d))
	}
	//dfns := strings.TrimSuffix(*dashboardFile, ".json")
	bdf := filepath.Base(d)
	bdfns := strings.TrimSuffix(bdf, ".json")
	if m == "" {
		m = fmt.Sprintf("%s.yaml", bdfns)
	}
	if n == "" {
		n = strings.Replace(bdfns, "_", "-", -1)
	}

	db := readDashboardJson(d)
	dt := db.Data
	dd, err := dt.Encode()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	var dp []byte
	if *compact {
		dp, err = dt.Encode()
	} else {
		dp, err = dt.EncodePretty()
	}
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	fmt.Sprintln(string(dd))
	cm := grafanaConfigMap{
		ApiVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: configMapMetadata{
			Name:      n,
			Namespace: *k8sNamespace,
			Labels: configMapMetadataLabels{
				GrafanaDashboard: "1",
			},
			Annotations: *k8sAnnotations,
		},
		Data: map[string]string{bdf: fmt.Sprintln(string(dp))},
	}
	md, err := yaml.Marshal(&cm)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	err = ioutil.WriteFile(m, md, 0666)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Error: %s could not be written (%s)", m, err))
	}
}

func cleanDir(dir, pattern string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			os.Remove(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func main() {
	kingpin.Parse()
	if *dashboardsDirGlob != "" && *manifestsDir != "" {
		if *cleanManifestsDir != "" {
			cleanDir(*manifestsDir, *cleanManifestsDir)
		}
		processDir(*dashboardsDirGlob, *manifestsDir)
	} else if *dashboardFile != "" && *manifestFile != "" {
		processFile(*dashboardFile, *manifestFile, *dashboardName)
	} else {
		log.Fatal().Msg(fmt.Sprintf("must set flags [(-f and -o) or (-d and -m)], -f: %s, -o: %s, -d: %s, -m: %s",
			*dashboardFile, *manifestFile, *dashboardsDirGlob, *manifestsDir))
	}
}
