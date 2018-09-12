package main

import (
	"fmt"
	"net"

	"github.com/Shopify/sarama"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
)

func checkKafkaReady(request internal.KafkaReadyParameters) (internal.KafkaReadyParameters, error) {
	response := request
	response.Ready = false

	// NOTE: All errors encountered here should be considered retriable.  Either
	// print them and return nil, or ensure the Step Function that calls this
	// function catches errors reported here.

	client, err := sarama.NewClient(
		[]string{net.JoinHostPort(request.InternalIPAddr, string(request.KafkaPort))},
		sarama.NewConfig(),
	)
	if err != nil {
		fmt.Println(err)
		return response, nil
	}
	topics, err := client.Topics()
	if err != nil {
		fmt.Println(err)
		return response, nil
	}
	for _, topic := range topics {
		partitions, err := client.Partitions(topic)
		if err != nil {
			fmt.Println(err)
			return response, nil
		}
		for _, partition := range partitions {
			replicas, err := client.Replicas(topic, partition)
			if err != nil {
				fmt.Println(err)
				return response, nil
			}
			isrs, err := client.InSyncReplicas(topic, partition)
			if err != nil {
				fmt.Println(err)
				return response, nil
			}
			fmt.Printf("Topic %s[%d]: %d replicas, %d ISRs\n", topic, partition, len(replicas), len(isrs))
			if len(replicas) != len(isrs) {
				fmt.Println("Not in sync - exiting")
				return response, nil
			}
		}
	}

	response.Ready = true
	return response, nil
}

func main() {
	lambda.Start(checkKafkaReady)
}
