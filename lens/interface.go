package lens

import (
	"context"
	"fmt"

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

type InstrumentedStore struct {
	store     adt.Store
	processor string
	method    string
	args      []interface{}
	gets      int
}

func NewInstrumentedStore(s adt.Store, processor string, method string, args ...interface{}) *InstrumentedStore {
	return &InstrumentedStore{
		store:     s,
		processor: processor,
		method:    method,
		args:      args,
	}
}

func (s *InstrumentedStore) Context() context.Context {
	return s.store.Context()
}

func (s *InstrumentedStore) Get(ctx context.Context, c cid.Cid, out interface{}) error {
	s.gets++
	return s.store.Get(ctx, c, out)
}

func (s *InstrumentedStore) Put(ctx context.Context, v interface{}) (cid.Cid, error) {
	return s.store.Put(ctx, v)
}

func (s *InstrumentedStore) Report() {
	log.Infow("InstrumentedStore", "processor", s.processor, "method", s.method, "args", fmt.Sprintf("%s", s.args), "gets", s.gets)
}
