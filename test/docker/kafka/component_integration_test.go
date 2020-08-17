// +build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/component/async/kafka/group"
	"github.com/stretchr/testify/suite"
)

type KafkaComponentTestSuite struct {
	suite.Suite
}

func TestKafkaComponentTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaComponentTestSuite))
}

func (s *KafkaComponentTestSuite) TestKafkaComponent_Success() {
	// Timeout the test if it takes too long
	timeoutTimer := testTimeout(s, 15*time.Second)

	// Test parameters
	numOfMessagesToSend := 100

	// Set up the kafka component
	actualSuccessfulMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(msg async.Message) error {
		var msgContent string
		err := msg.Decode(&msgContent)
		s.NoError(err)
		actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
		consumerWG.Done()
		return nil
	}
	component := s.NewComponent(successTopic, 3, processorFunc)

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		err := patron.New(successTopic, "0").
			WithComponents(component).
			Run(patronContext)
		s.Require().NoError(err)
		patronWG.Done()
	}()

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		s.Require().NoError(err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: successTopic, Value: sarama.StringEncoder(strconv.Itoa(i))})
			s.Require().NoError(err)
		}
		producerWG.Done()
	}()

	// Wait for both consumer and producer to finish processing all the messages.
	producerWG.Wait()
	consumerWG.Wait()

	// Verify all messages were processed in the right order
	expectedMessages := make([]string, numOfMessagesToSend)
	for i := 0; i < numOfMessagesToSend; i++ {
		expectedMessages[i] = strconv.Itoa(i + 1)
	}
	s.Equal(expectedMessages, actualSuccessfulMessages)

	// Shutdown Patron and wait for it to finish
	patronCancel()
	patronWG.Wait()

	// Make sure the timeout timer is stopped
	timeoutTimer.Stop()
}

func (s *KafkaComponentTestSuite) TestKafkaComponent_FailAllRetries() {
	// Timeout the test if it takes too long
	timeoutTimer := testTimeout(s, 15*time.Second)

	// Test parameters
	numOfMessagesToSend := 100
	errAtIndex := 50

	// Set up the kafka component
	actualSuccessfulMessages := make([]int, 0)
	actualNumOfRuns := int32(0)
	processorFunc := func(msg async.Message) error {
		var msgContentStr string
		err := msg.Decode(&msgContentStr)
		s.NoError(err)

		msgIndex, err := strconv.Atoi(msgContentStr)
		s.NoError(err)

		if msgIndex == errAtIndex {
			atomic.AddInt32(&actualNumOfRuns, 1)
			return errors.New("expected error")
		}
		actualSuccessfulMessages = append(actualSuccessfulMessages, msgIndex)
		return nil
	}
	numOfRetries := uint(3)
	component := s.NewComponent(failAllRetriesTopic, numOfRetries, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		s.Require().NoError(err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAllRetriesTopic, Value: sarama.StringEncoder(strconv.Itoa(i))})
			s.Require().NoError(err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component - no need for goroutine since we expect it to stop after the retries fail
	err := patron.New(failAllRetriesTopic, "0").
		WithComponents(component).
		Run(context.Background())
	s.Error(err)

	// Wait for the producer & consumer to finish
	producerWG.Wait()

	// Verify all messages were processed in the right order
	expectedMessages := make([]int, errAtIndex-1)
	for i := 0; i < errAtIndex-1; i++ {
		expectedMessages[i] = i + 1
	}
	s.Equal(expectedMessages, actualSuccessfulMessages)
	s.Equal(int32(numOfRetries+1), actualNumOfRuns)

	// Make sure the timeout timer is stopped
	timeoutTimer.Stop()
}

func (s *KafkaComponentTestSuite) TestKafkaComponent_FailOnceAndRetry() {
	// Timeout the test if it takes too long
	timeoutTimer := testTimeout(s, 15*time.Second)

	// Test parameters
	numOfMessagesToSend := 100

	// Set up the component
	didFail := int32(0)
	actualMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(msg async.Message) error {
		var msgContent string
		err := msg.Decode(&msgContent)
		s.NoError(err)

		if msgContent == "50" && atomic.CompareAndSwapInt32(&didFail, 0, 1) {
			return errors.New("expected error")
		}
		consumerWG.Done()
		actualMessages = append(actualMessages, msgContent)
		return nil
	}
	component := s.NewComponent(failAndRetryTopic, 3, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		s.Require().NoError(err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAndRetryTopic, Value: sarama.StringEncoder(strconv.Itoa(i))})
			s.Require().NoError(err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		err := patron.New(failAndRetryTopic, "0").
			WithComponents(component).
			Run(patronContext)
		s.Require().NoError(err)
		patronWG.Done()
	}()

	// Wait for the producer & consumer to finish
	producerWG.Wait()
	consumerWG.Wait()

	// Shutdown Patron and wait for it to finish
	patronCancel()
	patronWG.Wait()

	// Verify all messages were processed in the right order
	expectedMessages := make([]string, numOfMessagesToSend)
	for i := 0; i < numOfMessagesToSend; i++ {
		expectedMessages[i] = strconv.Itoa(i + 1)
	}
	s.Equal(expectedMessages, actualMessages)

	// Make sure the timeout timer is stopped
	timeoutTimer.Stop()
}

func (s *KafkaComponentTestSuite) NewComponent(name string, retries uint, processorFunc func(message async.Message) error) *async.Component {
	decode := func(data []byte, v interface{}) error {
		tmp := string(data)
		p := v.(*string)
		*p = tmp
		return nil
	}
	factory, err := group.New(
		name,
		name+"-group",
		[]string{name},
		[]string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)},
		kafka.Decoder(decode),
		kafka.Start(sarama.OffsetOldest))
	s.Require().NoError(err)

	cmp, err := async.New(name, factory, processorFunc).
		WithRetries(retries).
		WithRetryWait(200 * time.Millisecond).
		WithFailureStrategy(async.NackExitStrategy).
		Create()
	s.Require().NoError(err)

	return cmp
}

func testTimeout(s *KafkaComponentTestSuite, timeout time.Duration) *time.Timer {
	testName := s.T().Name()
	timeoutTimer := time.AfterFunc(timeout, func() {
		s.FailNowf("Timed out: test %s took more then %v to complete", testName, timeout)
	})
	return timeoutTimer
}
