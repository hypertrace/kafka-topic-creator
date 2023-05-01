package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"gopkg.in/yaml.v3"
)

type KafkaTopic struct {
	NumPartitions     int               `yaml:"partitions"`
	ReplicationFactor int               `yaml:"replicationFactor"`
	Configs           map[string]string `yaml:"configs"`
}

type Config struct {
	Address string                `yaml:"address"`
	Topics  map[string]KafkaTopic `yaml:"topics"`
}

func (config *Config) LoadConfiguration(file string) *Config {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return config
}

func ListTopics(a *kafka.AdminClient) []string {
	response, err := a.GetMetadata(nil, true, 60000)
	if err != nil {
		fmt.Printf("Failed to list topics: %v\n", err)
		os.Exit(1)
	}
	keys := make([]string, 0, len(response.Topics))
	for k := range response.Topics {
		keys = append(keys, k)
	}
	return keys
}

func GetTopicConfigs(a *kafka.AdminClient, topics []string) map[string]map[string]string {
	configs := make([]kafka.ConfigResource, 0, len(topics))
	for _, t := range topics {
		configs = append(configs, kafka.ConfigResource{Type: kafka.ResourceTopic, Name: t})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	duration, err := time.ParseDuration("60s")
	if err != nil {
		panic("ParseDuration(60s)")
	}

	results, err := a.DescribeConfigs(ctx, configs, kafka.SetAdminRequestTimeout(duration))
	if err != nil {
		fmt.Printf("Failed to describe topics: %v\n", err)
		os.Exit(1)
	}

	output := make(map[string]map[string]string)

	for _, result := range results {
		output[result.Name] = make(map[string]string)
		for _, entry := range result.Config {
			output[result.Name][entry.Name] = entry.Value
		}
	}
	return output
}

func CreateTopics(a *kafka.AdminClient, topics []kafka.TopicSpecification) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	duration, err := time.ParseDuration("60s")
	if err != nil {
		panic("ParseDuration(60s)")
	}

	results, err := a.CreateTopics(
		ctx,
		topics,
		kafka.SetAdminOperationTimeout(duration))
	if err != nil {
		fmt.Printf("Failed to create topic: %v\n", err)
		os.Exit(1)
	}

	error := false
	for _, result := range results {
		if result.Error.Code() != 0 {
			fmt.Printf("Failed to create topic: %s\n", result)
			error = true
		} else {
			fmt.Printf("successfully created topic: %s\n", result.Topic)
		}
	}
	if error {
		os.Exit(1)
	}

}

func AlterConfigs(a *kafka.AdminClient, resources []kafka.ConfigResource) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	duration, err := time.ParseDuration("60s")
	if err != nil {
		panic("ParseDuration(60s)")
	}
	results, err := a.AlterConfigs(ctx, resources, kafka.SetAdminRequestTimeout(duration))
	if err != nil {
		fmt.Printf("Failed to update topic configs: %v\n", err)
		os.Exit(1)
	}

	error := false
	for _, result := range results {
		// fmt.Printf("update topic %s\n", result.Name)
		// for _, entry := range result.Config {
		// 	fmt.Printf("%s: %s\n", entry.Name, entry.Value)
		// }
		if result.Error.Code() != 0 {
			fmt.Printf("Failed to update configs of topic %s\n", result)
			error = true
		} else {
			fmt.Printf("successfully updated topic: %s\n", result.Name)
		}
	}
	if error {
		os.Exit(1)
	}
}

func checkCleanupPolicy(value string) bool {
	if value == "[compact,delete]" || value == "[delete,compact]" || value == "delete,compact" || value == "compact,delete" {
		return true
	}
	return false
}

func checkConfigs(topicConfig map[string]string, newConfig map[string]string) bool {
	for key, newValue := range newConfig {
		existingValue, ok := topicConfig[key]
		if !ok {
			fmt.Printf("WARNING: config key %s not found\n", key)
			return true
		}
		if checkCleanupPolicy(newValue) {
			newValue = "compact,delete"
		}
		if checkCleanupPolicy(existingValue) {
			existingValue = "compact,delete"
		}
		if newValue != existingValue {
			fmt.Printf("WARNING: value for config %s does not match. %s != %s\n", key, existingValue, newValue)
			return true
		}
	}
	return false
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func main() {
	configpath := flag.String("configpath", "/opt/kafka/config.yaml", "config file path")
	flag.Parse()
	fmt.Printf("configpath: %s\n", *configpath)

	var config Config
	config.LoadConfiguration(*configpath)
	// fmt.Printf("config: %v\n\n", config)

	a, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": config.Address})
	if err != nil {
		fmt.Printf("Failed to create Admin client: %s\n", err)
		os.Exit(1)
	}
	topicList := ListTopics(a)

	newTopics := []kafka.TopicSpecification{}
	existingTopics := []string{}

	for topicName, topicConfig := range config.Topics {
		if contains(topicList, topicName) {
			existingTopics = append(existingTopics, topicName)
		} else {
			val, ok := topicConfig.Configs["cleanup.policy"]
			if ok {
				if checkCleanupPolicy(val) {
					topicConfig.Configs["cleanup.policy"] = "compact,delete"
				}
			}
			newTopics = append(newTopics, kafka.TopicSpecification{Topic: topicName, NumPartitions: topicConfig.NumPartitions, ReplicationFactor: topicConfig.ReplicationFactor, Config: topicConfig.Configs})
		}
	}

	fmt.Printf("Number of new topics: %d\n", len(newTopics))
	if len(newTopics) > 0 {
		CreateTopics(a, newTopics)
	}

	fmt.Printf("Number of existing topics: %d\n", len(existingTopics))
	if len(existingTopics) > 0 {
		updatedTopics := make([]kafka.ConfigResource, 0, len(existingTopics))
		existingTopicConfigs := GetTopicConfigs(a, topicList)
		for _, t := range existingTopics {
			fmt.Printf("checking config for topic %s\n", t)
			if checkConfigs(existingTopicConfigs[t], config.Topics[t].Configs) {
				configs := make([]kafka.ConfigEntry, 0, len(config.Topics[t].Configs))
				for k, v := range config.Topics[t].Configs {
					if checkCleanupPolicy(v) {
						v = "compact,delete"
					}
					configs = append(configs, kafka.ConfigEntry{Name: k, Value: v})
				}
				updatedTopics = append(updatedTopics, kafka.ConfigResource{Type: kafka.ResourceTopic, Name: t, Config: configs})
			}
		}
		fmt.Printf("Number of updated topics: %d\n", len(updatedTopics))
		if len(updatedTopics) > 0 {
			AlterConfigs(a, updatedTopics)
		}
	}
	// fmt.Println(output)
	a.Close()
}
