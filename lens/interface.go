package lens

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

type API interface {
	Store() adt.Store
	api.FullNode
	ComputeGasOutputs(gasUsed, gasLimit int64, baseFee, feeCap, gasPremium abi.TokenAmount) vm.GasOutputs
	GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*ExecutedMessage, error)
}

type APICloser func()

type APIOpener interface {
	Open(context.Context) (API, APICloser, error)
}

type ExecutedMessage struct {
	Cid           cid.Cid
	Height        abi.ChainEpoch
	Message       *types.Message
	Receipt       *types.MessageReceipt
	BlockHeader   *types.BlockHeader
	Blocks        []cid.Cid // blocks this message appeared in
	Index         uint64    // Message and receipt sequence in tipset
	FromActorCode cid.Cid   // code of the actor the message is from
	ToActorCode   cid.Cid   // code of the actor the message is to
}

var _ adt.Store = (*InstrumentedStore)(nil)

var log = logging.Logger("instrumentation")

var (
	mu                 sync.Mutex
	InstrumentedStores = []*InstrumentedStore{}
)

type InstrumentedStore struct {
	store     adt.Store
	processor string
	method    string
	args      []interface{}
	gets      map[cid.Cid]int
}

func NewInstrumentedStore(s adt.Store, processor string, method string, args ...interface{}) *InstrumentedStore {
	mu.Lock()
	defer mu.Unlock()

	is := &InstrumentedStore{
		store:     s,
		processor: processor,
		method:    method,
		args:      args,
		gets:      map[cid.Cid]int{},
	}

	InstrumentedStores = append(InstrumentedStores, is)

	return is
}

func (s *InstrumentedStore) Context() context.Context {
	return context.Background()
}

func (s *InstrumentedStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	s.gets[c] = s.gets[c] + 1
	return s.store.Get(context.Background(), c, out)
}

func (s *InstrumentedStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return s.store.Put(context.Background(), v)
}

func ReportInstrumentedStores() {
	for _, s := range InstrumentedStores {
		var maxN int
		var maxCid cid.Cid
		var total int

		for c, n := range s.gets {
			total += n
			if n > maxN {
				maxN = n
				maxCid = c
			}
		}
		log.Infow("InstrumentedStore", "processor", s.processor, "method", s.method, "args", fmt.Sprintf("%s", s.args), "total_gets", total, "unique_gets", len(s.gets), "most_requested_count", maxN, "most_requested_cid", maxCid)
	}

	InstrumentedStores = InstrumentedStores[:0]
}
