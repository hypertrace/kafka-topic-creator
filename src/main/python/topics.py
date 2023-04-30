from confluent_kafka.admin import AdminClient, NewTopic, ConfigResource
import json
import os

def list_topics(admin_client):
  topics = []
  for topic in admin_client.list_topics().topics:
    topics.append(topic)
  return topics

def get_topic_configs(admin_client, topics):
  output = {}
  config_resources = []
  for topic in topics:
    config_resources.append(ConfigResource('topic', topic))
  response = admin_client.describe_configs(config_resources)
  for key,value in response.items():
    output[key.name] = {}
    for k,v in value.result().items():
      output[key.name][k] = v.value
  return output

def create_topic(admin_client, topic_list):
  response = admin_client.create_topics(topic_list)
  for key,value in response.items():
    print('response while creating topic ' + key + ': ')
    print(value.result())

def alter_configs(admin_client, topic_list):
  response = admin_client.alter_configs(topic_list)
  for key,value in response.items():
    print('response while updating topic ' + key.name + ': ')
    print(value.result())

def check_cleanup_policy(value):
  if value == '[compact,delete]' or value == '[delete,compact]' or value == 'compact,delete' or value == 'delete,compact':
    return True
  return False

def check_configs(topic_config, new_config):
  for key,value in new_config.items():
    if key not in topic_config:
      print('WARNING: config ' + key + ' not found.')
      return False
    if key == 'cleanup.policy':
      if check_cleanup_policy(value):
        value = 'COMPACT_DELETE'
      if check_cleanup_policy(topic_config[key]):
        topic_config[key] = 'COMPACT_DELETE'
    if str(value) != str(topic_config[key]):
      print('WARNING: value of config ' + key + ' does not match: ' + str(value) + ' != ' + str(topic_config[key]))
      return False
  return True

def main(filepath):
  with open(filepath) as fd:
    input = json.load(fd)
  address = input['address']
  topics = input['topics']
  admin_client = AdminClient({"bootstrap.servers": address})
  topic_list = list_topics(admin_client)
  new_topics = []
  existing_topics = []
  for key,value in topics.items():
    if key in topic_list:
      existing_topics.append(key)
    else:
      print('new topic: ' + key)
      if 'cleanup.policy' in value['configs'] and check_cleanup_policy(value['configs']['cleanup.policy']):
        value['configs']['cleanup.policy'] = 'compact,delete'
      new_topic = NewTopic(key, num_partitions=int(value['partitions']), replication_factor=int(value['replicationFactor']), config={k:str(value['configs'][k]) for k in value['configs']})
      new_topics.append(new_topic)
  print('new_topics: ' + str(len(new_topics)))
  if len(new_topics) > 0:
    create_topic(admin_client, new_topics)
  print('existing_topics: ' + str(len(existing_topics)))
  if len(existing_topics) > 0:
    configs = get_topic_configs(admin_client, existing_topics)
    updated_topics = []
    for topic in existing_topics:
      print('checking configs of topic ' + topic)
      if not check_configs(configs[topic], topics[topic]['configs']):
        if 'cleanup.policy' in topics[topic]['configs'] and check_cleanup_policy(topics[topic]['configs']['cleanup.policy']):
            topics[topic]['configs']['cleanup.policy'] = 'compact,delete'
        updated_topics.append(ConfigResource('topic', topic, topics[topic]['configs']))
    print('updated_topics: ' + str(len(updated_topics)))
    if len(updated_topics) > 0:
      alter_configs(admin_client, updated_topics)

if __name__ == '__main__':
  filepath = os.environ.get('INPUT_FILE', '/opt/kafka/topics.json')
  main(filepath)
