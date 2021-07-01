# gdb2cm

Convert a grafana dashboard json files into a k8s ConfigMap.

## build

`go build`

## run

```bash
usage: main --file.dashboard=FILE.DASHBOARD [<flags>]

Flags:
      --help               Show context-sensitive help (also try --help-long and --help-man).
  -f, --file.dashboard=FILE.DASHBOARD
                           Grafana dashboard JSON file to convert.
  -o, --file.output=""     Output file for the dashboard configmap.
  -c, --file.compact       Output file with compact JSON embedded in ConfigMap.
  -n, --dashboard.name=""  Dashboard configmap name. (Default: dashboard file basename)
  -a, --k8s.annotations=K8S.ANNOTATIONS ...
                           Add an annotation to add the dashboard configmap (key=value)
  -N, --k8s.namespace="monitoring"
                           kubernetes namespace for the configmap.
```
