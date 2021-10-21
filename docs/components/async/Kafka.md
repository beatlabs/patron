# Kafka Consumer

The package contains two sub-packages:

- `simple` which connects to each partition and consumes messages from each partition independently
- `group` which uses consumer groups in order to get messages

Both of the packages contain the factory and consumer implementation.
It is necessary to provide Sarama configuration when creating these consumers; you can use `v2.DefaultConsumerSaramaConfig` for sane defaults.

There is a special feature in the simple package which allows the consumer to go back a specific amount of time in each partition.  
This allows us to consume the messages from an approximate time onwards.
