/*
 * Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.
 *
 * SoftlaneIT licenses this file to you under the Apache License,
 * Version 2.0 (the "LICENSE"); you may not use this file except
 * in compliance with the LICENSE.
 * You may obtain a copy of the LICENSE at
 *
 * https://softlaneit.com/LICENSE.txt
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the LICENSE is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the LICENSE for the
 * specific language governing permissions and limitations
 * under the LICENSE.
 */

package events

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaPublisher is the kafka-go-backed implementation of Publisher.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// NewKafka returns a KafkaPublisher that writes to brokers.
func NewKafka(brokers []string) *KafkaPublisher {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{},
		BatchTimeout:           10 * time.Millisecond,
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
	}
	return &KafkaPublisher{writer: w}
}

// Publish sends a message to topic.
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("kafka publish to %s: %w", topic, err)
	}
	return nil
}

// Close flushes and closes the writer.
func (p *KafkaPublisher) Close() error { return p.writer.Close() }

// NoopPublisher discards all events.
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, _ string, _ []byte, _ []byte) error { return nil }
func (NoopPublisher) Close() error                                                   { return nil }
