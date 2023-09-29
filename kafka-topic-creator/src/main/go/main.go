package main

import (
	"context"
	"flag"
	"log"
	"os"
	"slices"
	"strconv"
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
	Address                        string                `yaml:"address"`
	Topics                         map[string]KafkaTopic `yaml:"topics"`
	MinValueOverrideForTopicConfig map[string]int64      `yaml:"minValueOverrideForTopicConfig"`
}

const timeoutDuration = 60 * time.Second

var adminRequestTimeout = kafka.SetAdminRequestTimeout(timeoutDuration)

func (config *Config) LoadConfiguration(file string) *Config {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		log.Panic(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		log.Panic(err.Error())
	}
	return config
}

func ListTopics(a *kafka.AdminClient) []string {
	response, err := a.GetMetadata(nil, true, int(timeoutDuration.Milliseconds()))
	if err != nil {
		log.Panicf("Failed to list topics: %v\n", err)
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

	results, err := a.DescribeConfigs(context.Background(), configs, adminRequestTimeout)
	if err != nil {
		log.Panicf("Failed to describe topics: %v\n", err)
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
	results, err := a.CreateTopics(
		context.Background(),
		topics, adminRequestTimeout)
	if err != nil {
		log.Panicf("Failed to create topic: %v\n", err)
	}

	failed := false
	for _, result := range results {
		if result.Error.Code() != 0 {
			log.Printf("Failed to create topic: %s\n", result)
			failed = true
		} else {
			log.Printf("successfully created topic: %s\n", result.Topic)
		}
	}
	if failed {
		log.Panic("failed to create topics")
	}

}

func AlterConfigs(a *kafka.AdminClient, resources []kafka.ConfigResource) {
	results, err := a.AlterConfigs(context.Background(), resources, adminRequestTimeout)
	if err != nil {
		log.Panicf("Failed to update topic configs: %v\n", err)
	}

	failed := false
	for _, result := range results {
		if result.Error.Code() != 0 {
			log.Printf("Failed to update configs of topic %s\n", result)
			failed = true
		} else {
			log.Printf("successfully updated topic: %s\n", result.Name)
		}
	}
	if failed {
		log.Panic("failed to update topics")
	}
}

func checkCleanupPolicy(value string) bool {
	if value == "[compact,delete]" || value == "[delete,compact]" || value == "delete,compact" || value == "compact,delete" {
		return true
	}
	return false
}

func getResolvedConfig(topic string, topicConfig, newConfig map[string]string, minValueOverrideForTopicConfig map[string]int64) (map[string]string, bool) {
	resolvedConfig := make(map[string]string) // new config applied with overrides
	needsUpdate := false
	for key, newValue := range newConfig {
		existingValue, ok := topicConfig[key]
		if !ok {
			log.Printf("WARNING[%s]: config key %s not found in existing config\n", topic, key)
			needsUpdate = true
			break
		}
		if checkCleanupPolicy(existingValue) {
			existingValue = "compact,delete"
		}
		if checkCleanupPolicy(newValue) {
			newValue = "compact,delete"
		}
		if minOverrideValue, isInOverride := minValueOverrideForTopicConfig[key]; isInOverride {
			inNewConfigValue, err := strconv.ParseInt(newValue, 10, 64)
			if err != nil {
				log.Panicf("new config for %s key for topic %s is not parseable as int: %s", key, topicConfig, newValue)
			}
			if inNewConfigValue < minOverrideValue {
				log.Printf("MIN_OVERRIDING[%s]: config key %s in new config is %s, lower than min override %d\n", topic, key, newValue, minOverrideValue)
				newValue = strconv.FormatInt(minOverrideValue, 10)
			}
		}
		if newValue != existingValue {
			log.Printf("WARNING[%s]: value for config %s does not match. %s != %s\n", topic, key, existingValue, newValue)
			needsUpdate = true
			break
		}
		resolvedConfig[key] = newValue
	}
	return resolvedConfig, needsUpdate
}

func main() {
	configpath := flag.String("configpath", "/opt/kafka/config.yaml", "config file path")
	flag.Parse()
	log.Printf("configpath: %s\n", *configpath)

	var config Config
	config.LoadConfiguration(*configpath)

	a, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": config.Address})
	defer a.Close()
	if err != nil {
		log.Panicf("Failed to create Admin client: %s\n", err)
	}
	topicList := ListTopics(a)

	var newTopics []kafka.TopicSpecification
	var existingTopics []string

	for topicName, topicConfig := range config.Topics {
		if slices.Contains(topicList, topicName) {
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

	existingTopicConfigs := GetTopicConfigs(a, topicList)

	log.Printf("Number of new topics: %d\n", len(newTopics))
	if len(newTopics) > 0 {
		CreateTopics(a, newTopics)
	}

	log.Printf("Number of existing topics: %d\n", len(existingTopics))
	if len(existingTopics) > 0 {
		updatedTopics := make([]kafka.ConfigResource, 0, len(existingTopics))
		for _, t := range existingTopics {
			log.Printf("checking config for topic %s\n", t)
			if resolvedConfig, needsUpdate := getResolvedConfig(t, existingTopicConfigs[t], config.Topics[t].Configs, config.MinValueOverrideForTopicConfig); needsUpdate {
				configs := make([]kafka.ConfigEntry, 0, len(resolvedConfig))
				for k, v := range resolvedConfig {
					configs = append(configs, kafka.ConfigEntry{Name: k, Value: v})
				}
				updatedTopics = append(updatedTopics, kafka.ConfigResource{Type: kafka.ResourceTopic, Name: t, Config: configs})
			}
		}
		log.Printf("Number of updated topics: %d\n", len(updatedTopics))
		if len(updatedTopics) > 0 {
			AlterConfigs(a, updatedTopics)
		}
	}
}
