package syncer

import (
	"sync"
)

type Syncer interface {
	//semaphore Proberen
	P() bool
	//semaphore Verhogen
	V()

	//IfOpen executes callback if the syncer is open.
	//and generally avoid it. This package is about making the caller
	//a goroutine without risk; not call goroutine inside synced operations
	IfOpen(Callback, []interface{}) Syncer
	IfClosed(Callback, []interface{}) Syncer
	// Clone makes a copy of Syncer. A closed syncer cannot be used any more.
	// If you want to use a syncer with same characteristics, clone it
	Clone() Syncer
	//Open starts the syncer.
	//Note that, a soon as syncer is created, IfClosed(..) will be able to operate,
	//even if syncer not started.
	Open()
	//Close stops the syncer.
	Close()
	//Done returns when syncer is fully stopped. If called before syncer started,
	//it will return immdiately.
	WaitClosed()
	//Ready blocks, and continues when the syncer has finished Open() operartions.
	WaitOpened()

	OpenedChannel() chan struct{}
	ClosedChannel() chan struct{}
}

type Callback func(v []interface{})

type SyncerImpl struct {
	poolSize  int
	pool      chan bool
	done      chan struct{}
	ready     chan struct{}
	poolMutex *sync.RWMutex
}

func NewSyncer(poolSize int) Syncer {
	if 0 > poolSize {
		poolSize = 1
	}
	s := &SyncerImpl{
		poolSize:  poolSize,
		poolMutex: new(sync.RWMutex),
		//ready & done are both the intransitive parts of a syncer
		ready: make(chan struct{}),
		done:  make(chan struct{}),
	}

	return s
}

func (s *SyncerImpl) Open() {
	defer close(s.ready)
	s.poolMutex.Lock()
	defer s.poolMutex.Unlock()

	s.pool = make(chan bool, s.poolSize)
	for i := 0; i < s.poolSize; i++ {
		s.pool <- true
	}
}

func (s *SyncerImpl) WaitOpened() {
	<-s.ready
}

func (s *SyncerImpl) Close() {
	s.poolMutex.RLock()
	defer s.poolMutex.RUnlock()

	if s.pool == nil {
		close(s.done)
		return
	}
	go func() {
		for i := 0; i < s.poolSize; i++ {
			<-s.pool
		}

		s.poolMutex.Lock()
		close(s.pool)
		s.pool = nil
		s.poolMutex.Unlock()
		close(s.done)
	}()
}

func (s *SyncerImpl) Clone() Syncer {
	s2 := &SyncerImpl{
		poolSize: s.poolSize,
		pool:     make(chan bool, s.poolSize),
	}
	return s2
}

func (s *SyncerImpl) IfOpen(cb Callback, args []interface{}) Syncer {
	s.poolMutex.RLock()
	defer s.poolMutex.RUnlock()

	if s.pool == nil {
		return s
	}

	r := s.P()
	defer s.V()

	if !r {
		return s
	}

	cb(args)

	return s
}

func (s *SyncerImpl) IfClosed(cb Callback, args []interface{}) Syncer {
	s.poolMutex.RLock()
	defer s.poolMutex.RUnlock()

	if s.pool == nil {
		cb(args)
		return s
	}

	r := s.P()
	defer s.V()

	if r {
		return s
	}
	cb(args)

	return s
}

func (s *SyncerImpl) P() bool {
	return <-s.pool
}

func (s *SyncerImpl) V() {
	s.pool <- true
}

func (s *SyncerImpl) WaitClosed() {
	s.poolMutex.RLock()
	if s.pool == nil {
		return
	}
	s.poolMutex.RUnlock()
	<-s.done
}

func (s *SyncerImpl) ClosedChannel() chan struct{} {
	return s.done
}

func (s *SyncerImpl) OpenedChannel() chan struct{} {
	return s.ready
}
