package syncer

type Syncer interface {
	//semaphore Proberen
	P() bool
	//semaphore Verhogen
	V()

	//in theory, callbacks should be careful if including goroutine,
	//and generally avoid it. This package is about making the caller
	//a goroutine without risk; not call goroutine inside synced operations
	IfOpen(Callback, []interface{})
	IfClosed(Callback, []interface{})

	// a closed syncer cannot be used any more.
	// if you want to use a syncer with same characteristics, clone it
	Clone() Syncer

	Close()
	Done()
	DoneChannel() chan struct{}
}
type Callback func(v ...interface{})

type SyncerImpl struct {
	poolSize int
	pool     chan bool
	done     chan struct{}
}

func NewSyncer(poolSize int) Syncer {
	if 0 > poolSize {
		poolSize = 1
	}
	s := &SyncerImpl{
		poolSize: poolSize,
		pool:     make(chan bool, poolSize),
	}

	for i := 0; i < poolSize; i++ {
		s.pool <- true
	}
	return s
}

func (s *SyncerImpl) Close() {
	go func(poolSize int, pool chan bool, done chan struct{}) {
		for i := 0; i < poolSize; i++ {
			<-pool
		}
		close(pool)
		close(done)
	}(s.poolSize, s.pool, s.done)
}

func (s *SyncerImpl) Clone() Syncer {
	s2 := &SyncerImpl{
		poolSize: s.poolSize,
		pool:     make(chan bool, s.poolSize),
	}
	return s2
}

func (s *SyncerImpl) IfOpen(cb Callback, args []interface{}) {
	r := s.P()
	if !r {
		return
	}
	cb(args)
	s.V()
}

func (s *SyncerImpl) IfClosed(cb Callback, args []interface{}) {
	r := s.P()
	if r {
		return
	}
	cb(args)
}

func (s *SyncerImpl) P() bool {
	return <-s.pool
}

func (s *SyncerImpl) V() {
	s.pool <- true
}

func (s *SyncerImpl) Done() {
	<-s.done
}

func (s *SyncerImpl) DoneChannel() chan struct{} {
	return s.done
}
