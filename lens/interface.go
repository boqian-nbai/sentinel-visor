package lens

import (
	"context"
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
	id        string
	gets      int
}

func NewInstrumentedStore(s adt.Store, processor string, method string, id string) *InstrumentedStore {
	mu.Lock()
	defer mu.Unlock()

	is := &InstrumentedStore{
		store:     s,
		processor: processor,
		method:    method,
		id:        id,
	}

	InstrumentedStores = append(InstrumentedStores, is)

	return is
}

func (s *InstrumentedStore) Context() context.Context {
	return context.Background()
}

func (s *InstrumentedStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	s.gets++
	return s.store.Get(context.Background(), c, out)
}

func (s *InstrumentedStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return s.store.Put(context.Background(), v)
}

func ReportInstrumentedStores(ts *types.TipSet) {
	log.Infow("InstrumentedStore", "height", ts.Height(), "stateroot", ts.ParentState(), "tipset", ts.String())
	for _, s := range InstrumentedStores {
		log.Infow("InstrumentedStore", "processor", s.processor, "method", s.method, "id", s.id, "gets", s.gets)
	}

	InstrumentedStores = InstrumentedStores[:0]
}
