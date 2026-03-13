package broadcaster

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscribe_ReturnsNonNilChannel(t *testing.T) {
	b := NewBroadcaster(nil)
	ch, unsubscribe := b.Subscribe()
	defer unsubscribe()

	assert.NotNil(t, ch, "Subscribe should return a non-nil channel")
	assert.Equal(t, subscriberBufferSize, cap(ch), "Channel should have the correct buffer size")
}

func TestSubscribe_BroadcastDeliversEventToChannel(t *testing.T) {
	b := NewBroadcaster(nil)
	ch, unsubscribe := b.Subscribe()
	defer unsubscribe()

	eventName := "testEvent"
	payload := map[string]string{"key": "value"}

	b.Broadcast(eventName, payload)

	select {
	case event := <-ch:
		assert.Equal(t, eventName, event.Name, "Event name should match")
		expectedPayload, _ := json.Marshal(payload)
		assert.Equal(t, expectedPayload, event.Payload, "Event payload should match")
	default:
		t.Fatal("Expected an event to be delivered to the channel, but none was received")
	}
}

func TestSubscribe_UnsubscribeClosesChannelAndRemovesSubscriber(t *testing.T) {

	b := NewBroadcaster(nil)
	ch, unsubscribe := b.Subscribe()
	unsubscribe()

	// Assert channel is closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed after unsubscribe")

	// Broadcast an event and assert nothing is delivered to the channel
	b.Broadcast("testEvent", map[string]string{"key": "value"})
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "No event should be delivered to the channel after unsubscribe")
	default:
		// No event received, as expected
	}
}

func TestSubscribe_CallingUnsubscribeTwiceDoesNotPanic(t *testing.T) {

	b := NewBroadcaster(nil)
	_, unsubscribe := b.Subscribe()
	unsubscribe()
	unsubscribe()
}

func TestSubscribe_TwoSubscribersAreIndependent(t *testing.T) {
	b := NewBroadcaster(nil)
	ch1, unsub1 := b.Subscribe()
	ch2, unsub2 := b.Subscribe()
	defer unsub2() // keep ch2 active for this test

	unsub1() // only ch1 unsubscribed before broadcast

	eventName := "testEvent"
	payload := map[string]string{"key": "value"}
	b.Broadcast(eventName, payload)

	select {
	case _, ok := <-ch1:
		assert.False(t, ok, "No event should be delivered to ch1 after unsubscribing")
	default:
		// acceptable too; channel may already be closed
	}

	select {
	case event, ok := <-ch2:
		assert.True(t, ok, "ch2 should still be open and receive events")
		assert.Equal(t, eventName, event.Name, "Event name should match for ch2")
		expectedPayload, _ := json.Marshal(payload)
		assert.Equal(t, expectedPayload, event.Payload, "Event payload should match for ch2")
	default:
		t.Fatal("Expected an event to be delivered to ch2, but none was received")
	}
}
