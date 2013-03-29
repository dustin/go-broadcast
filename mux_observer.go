package broadcast

type taggedObservation struct {
	sub *subObserver
	ob  interface{}
}

const (
	register = iota
	unregister
	purge
)

type taggedRegReq struct {
	sub     *subObserver
	ch      chan<- interface{}
	regType int
}

// A MuxObserver multiplexes several streams of observations onto a
// single delivery goroutine.
type MuxObserver struct {
	subs  map[*subObserver]map[chan<- interface{}]bool
	reg   chan taggedRegReq
	input chan taggedObservation
}

// Get a new MuxObserver.
//
// qlen is the size of the channel buffer for observations sent into
// the mux observer and reglen is the size of the channel buffer for
// registration/unregistration events.
func NewMuxObserver(qlen, reglen int) *MuxObserver {
	rv := &MuxObserver{
		subs:  map[*subObserver]map[chan<- interface{}]bool{},
		reg:   make(chan taggedRegReq, reglen),
		input: make(chan taggedObservation, qlen),
	}
	go rv.run()
	return rv
}

// Shut down this mux observer.
func (b *MuxObserver) Close() error {
	close(b.reg)
	return nil
}

func (b *MuxObserver) broadcast(to taggedObservation) {
	for ch := range b.subs[to.sub] {
		ch <- to.ob
	}
}

func (b *MuxObserver) doReg(tr taggedRegReq) {
	m, exists := b.subs[tr.sub]
	if !exists {
		m = map[chan<- interface{}]bool{}
		b.subs[tr.sub] = m
	}
	m[tr.ch] = true
}

func (b *MuxObserver) doUnreg(tr taggedRegReq) {
	m, exists := b.subs[tr.sub]
	if exists {
		delete(m, tr.ch)
		if len(m) == 0 {
			delete(b.subs, tr.sub)
		}
	}
}

func (b *MuxObserver) handleReg(tr taggedRegReq) {
	switch tr.regType {
	case register:
		b.doReg(tr)
	case unregister:
		b.doUnreg(tr)
	case purge:
		delete(b.subs, tr.sub)
	}
}

func (b *MuxObserver) run() {
	for {
		select {
		case tr, ok := <-b.reg:
			if ok {
				b.handleReg(tr)
			} else {
				return
			}
		default:
			select {
			case to := <-b.input:
				b.broadcast(to)
			case tr, ok := <-b.reg:
				if ok {
					b.handleReg(tr)
				} else {
					return
				}
			}
		}
	}
}

// Create a new sub-broadcaster from this MuxObserver.
func (m *MuxObserver) Sub() Broadcaster {
	return &subObserver{m}
}

type subObserver struct {
	mo *MuxObserver
}

func (s *subObserver) Register(ch chan<- interface{}) {
	s.mo.reg <- taggedRegReq{s, ch, register}
}

func (s *subObserver) Unregister(ch chan<- interface{}) {
	s.mo.reg <- taggedRegReq{s, ch, unregister}
}

func (s *subObserver) Close() error {
	s.mo.reg <- taggedRegReq{s, nil, purge}
	return nil
}

func (s *subObserver) Submit(ob interface{}) {
	s.mo.input <- taggedObservation{s, ob}
}
