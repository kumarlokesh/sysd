package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/consumer"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/coordinator"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/producer"
)

func main() {
	// Initialize components
	coord := coordinator.NewCoordinator()
	messageLog := common.NewMessageLog()

	// Create a producer
	prod := producer.NewProducer("example-producer-1", coord, messageLog)

	// Create a consumer group
	consumer1 := consumer.NewConsumer("example-group-1", messageLog)

	// Subscribe to a topic
	topic := common.Topic("test-topic")
	partition := common.Partition(0)
	if err := consumer1.Subscribe(topic, partition); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Start a goroutine to consume messages
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msgs, err := consumer1.Poll(10)
				if err != nil {
					log.Printf("Error polling messages: %v", err)
					time.Sleep(time.Second)
					continue
				}

				for _, msg := range msgs {
					fmt.Printf("Consumed message: Topic=%s, Partition=%d, Offset=%d, Key=%s, Value=%s\n",
						msg.Topic, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value))
				}

				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Example 1: Successful transaction
	fmt.Println("\n=== Example 1: Successful Transaction ===")
	err := prod.BeginTransaction(30 * time.Second)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = prod.Send(topic, partition, []byte("key1"), []byte("value1"))
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	_, err = prod.Send(topic, partition, []byte("key2"), []byte("value2"))
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("Committing transaction...")
	err = prod.CommitTransaction()
	if err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	// Wait for messages to be consumed
	time.Sleep(1 * time.Second)

	// Example 2: Aborted transaction
	fmt.Println("\n=== Example 2: Aborted Transaction ===")
	err = prod.BeginTransaction(30 * time.Second)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = prod.Send(topic, partition, []byte("key3"), []byte("value3"))
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("Aborting transaction...")
	err = prod.AbortTransaction()
	if err != nil {
		log.Fatalf("Failed to abort transaction: %v", err)
	}

	// Wait for any final messages
	time.Sleep(1 * time.Second)
	cancel()
}
