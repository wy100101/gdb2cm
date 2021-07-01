package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/rs/zerolog/log"
	"github.com/wy100101/gdb2cm/pkg"
)

var (
	dashboardFile  = kingpin.Flag("file.dashboard", "Grafana dashboard JSON file to convert.").Short('f').Required().ExistingFile()
	manifestFile   = kingpin.Flag("file.output", "Output file for the dashboard configmap.").Short('o').Default("").String()
	compact        = kingpin.Flag("file.compact", "Output file with compact JSON embedded in ConfigMap.").Short('c').Default("false").Bool()
	dashboardName  = kingpin.Flag("dashboard.name", "Dashboard configmap name. (Default: dashboard file basename)").Short('n').Default("").String()
	k8sAnnotations = kingpin.Flag("k8s.annotations", "Add an annotation to add the dashboard configmap (key=value)").Short('a').StringMap()
	k8sNamespace   = kingpin.Flag("k8s.namespace", "kubernetes namespace for the configmap.").Short('N').Default("monitoring").String()
)

func main() {
	log.Logger = log.With().Caller().Logger()
	kingpin.Parse()
	err := gdb2cm.ProcessDashboardFile(*dashboardFile, *manifestFile, *k8sNamespace, *dashboardName, *compact, k8sAnnotations)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
