# kafka-topic-creator
Helm chart to create Kafka topic(s) via helm hook
test
## Usage
If any of the Kafka producer jobs wants to create Kafka topic(s) via a helm hook,
Add the following to `parentchart/Chart.yaml`
```$yaml
dependencies:
  - name: kafka-topic-creator
    repository: "https://hypertrace-helm-charts.storage.googleapis.com"
    version: 0.1.0
```

And, pass the values in `parentchart/values.yaml` file.

```$yaml
kafka-topic-creator:
  jobName: test-topic-creator
  helmHook: pre-install,pre-upgrade
  kafka:
    topics:
      - name: test-topic
        replicationFactor: 2
        partitions: 8
        configs:
          - retention.bytes=4294967296
          - retention.ms=259200000
  zookeeper:
    address: zookeeper:2181
  imagePullSecrets:
    - name: regcred
```
