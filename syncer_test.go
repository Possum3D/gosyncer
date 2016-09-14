package syncer

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncer(t *testing.T) {
	stream := make(chan string)
	mySyncer := NewSyncer(10)
	streamDone := make(chan struct{})

	msgs := make(chan []string, 1)

	//we can verify that a send operation before processor is setup will not be fatal:
	//it will simply call IfClosed func
	DummySend("not sent", mySyncer, stream)

	//Processor function
	go func(msgs chan []string, mySyncer Syncer, stream chan string, streamDone chan struct{}) {
		count := 0
		var buffer []string
		for {
			select {
			case msg := <-stream:
				buffer = append(buffer, msg)
			case <-mySyncer.ClosedChannel():
				close(stream)
				close(streamDone)
				msgs <- buffer
				close(msgs)
				return
			default:
				if count == 0 {
					//We open the syncer when processor is full setup, in particular
					//when stream is ready, as it is the output of our IfOpen func.
					mySyncer.Open()
					count += 1
				}
			}
		}
	}(msgs, mySyncer, stream, streamDone)

	//We wait for syncer to be opened, before sending data for real, that is,
	//data that will be executed with the IfOpen func
	mySyncer.WaitOpened()

	//At this stage, we know for sure that syncer is ready

	//We send data with multiple // threads
	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, mySyncer Syncer, stream chan string) {
			DummySend("sent", mySyncer, stream)
			wg.Done()
		}(wg, mySyncer, stream)
	}

	wg.Wait()

	//We close the syncer: it should have as effect to close properly the stream too,
	//per design of the processor. this is a common use-case: a cascade of channels
	//must be closed properly.
	mySyncer.Close()
	//At this stage, we know for sure that syncer was stopped.(processor returned)

	<-streamDone
	//At this stage, we know that stream was totally closed

	//We can verify that a send operation done after closing has the same effect as before:
	//IfClosed func is executed
	DummySend("not sent", mySyncer, stream)

	messages := <-msgs

	assert.Equal(t, 10, len(messages))
	for _, msg := range messages {
		assert.Equal(t, "sent", msg)
	}
}

// DummySend is a typical example of the use-case: it would be
// an interface method used by a server to send messages to a processor. The server
// must not be blocked, even if processor not fully ready, not started, or already
//closed ... It must always be able to continue.
func DummySend(message string, s Syncer, outputStream chan string) {
	var args []interface{}
	args = []interface{}{message, outputStream}

	s.IfOpen(func(args []interface{}) {
		mI := args[0]
		cI := args[1]
		m := mI.(string)
		c := cI.(chan string)
		c <- m
	}, args)

	s.IfClosed(func([]interface{}) {
		return
	}, args)
}
