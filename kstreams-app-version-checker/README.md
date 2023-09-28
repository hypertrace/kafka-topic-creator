# kstreams-app-version-checker
Helm chart is used for comparing the kafka streams application current version with the new version before upgradation. If there is a change in major version then it will scale down the kubernetes workload replicas to 0 and sleep for a fixed interval of time (greater than `consumer.session.timeout.ms`). This will allow seamless upgrade in case of incompatible change in kafka streams topology.  

This helm chart is designed to be used as a sub-chart of main application helm chart.

## Usage
Make the following changes in the application helm chart:

1. Add this chart to the dependencies list of the application chart:
   ```yaml
    dependencies:
      - name: kstreams-app-version-checker
        repository: https://storage.googleapis.com/hypertrace-helm-charts
        version: 1.0.1
        condition: kstreams-app-version-checker.enabled
   ```

2. Create a new file in `templates` folder with the following content:
   ```
   {{- if (index .Values "kstreams-app-version-checker" "enabled") }}
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: kvck-{{ .Release.Name }}
     labels:
       release: {{ .Release.Name }}
     annotations:
       {{- toYaml (index .Values "kstreams-app-version-checker" "annotations") | nindent 4 }}
   data:
     version: {{ default .Chart.AppVersion .Values.image.tagOverride | quote }}
   {{- end }}
   ```
   This will create a ConfigMap with the version of the application helm chart. This ConfigMap will be mounted into the pod so that job can compare the existing version with the new version.

3. Update the `values.yaml` file:
   ```yaml
   kstreams-app-version-checker:
     enabled: true
     workloads:
       - name: span-normalizer
         type: deployment
         container: span-normalizer
       - name: raw-spans-grouper
         type: statefulset
         container: raw-spans-grouper
   ```
