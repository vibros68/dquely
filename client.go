package dquely

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/dgo/v250"
	"github.com/dgraph-io/dgo/v250/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ErrDuplicate is returned when a conditional mutation does not fire because a
// unique constraint already matched an existing node. Test for it with errors.Is.
var ErrDuplicate = errors.New("dquely: duplicate unique constraint")

// ErrNotFound is returned by Query[T].First when no node matches the filter.
// Test for it with errors.Is.
var ErrNotFound = errors.New("dquely: not found")

type Config struct {
	DNS       string `mapstructure:"dns"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	Namespace uint64 `mapstructure:"namespace"`
}

type Dgo struct {
	DG    *dgo.Dgraph
	Debug bool
	// Logf, when set, receives debug output instead of the default fmt.Printf.
	// It matches the fmt.Printf signature so a standard logger can be adapted:
	//   d.Logf = log.Printf
	Logf func(format string, args ...any)
}

// logf routes debug output through the injected Logf callback, falling back to
// fmt.Printf when none is set.
func (d *Dgo) logf(format string, args ...any) {
	if d.Logf != nil {
		d.Logf(format, args...)
		return
	}
	fmt.Printf(format, args...)
}

// NewClient creates a Dgraph client using ACL credentials and verifies
// connectivity. Call Close() when the client is no longer needed.
func NewClient(cfg Config) (*Dgo, error) {
	if cfg.Username == "" || cfg.Password == "" {
		return nil, errors.New("dgo: username and password are required")
	}
	return newClient(cfg, dgo.WithACLCreds(cfg.Username, cfg.Password))
}

// NewClientInsecure creates a Dgraph client without ACL credentials, for local
// or development clusters that do not have access control enabled.
func NewClientInsecure(cfg Config) (*Dgo, error) {
	return newClient(cfg)
}

func newClient(cfg Config, extra ...dgo.ClientOption) (*Dgo, error) {
	opts := []dgo.ClientOption{
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}
	opts = append(opts, extra...)

	if cfg.Namespace != 0 {
		opts = append(opts, dgo.WithNamespace(cfg.Namespace))
	}

	dg, err := dgo.NewClient(cfg.DNS, opts...)
	if err != nil {
		return nil, fmt.Errorf("dgo: connect to %s: %w", cfg.DNS, err)
	}

	return &Dgo{DG: dg}, nil
}

// Close releases all underlying gRPC connections.
func (d *Dgo) Close() {
	d.DG.Close()
}

func (d *Dgo) SetSchema(ctx context.Context, schema string) error {
	op := &api.Operation{
		Schema: schema,
	}
	return d.DG.Alter(ctx, op)
}

func (d *Dgo) debugMutation(query string, mu *api.Mutation) {
	d.logf("query: %s\n", query)
	d.logf("condition: %s\n", mu.Cond)
	d.logf("SetNquads: %s\n", mu.SetNquads)
	d.logf("DelNquads: %s\n", mu.DelNquads)
}

// txnRunner is the subset of a Dgraph transaction needed to execute a request.
// Both *dgo.Txn (via d.DG.NewTxn()) and an open *Txn satisfy it.
type txnRunner interface {
	Do(ctx context.Context, req *api.Request) (*api.Response, error)
}

// execMutate builds and runs a mutation on the given runner. commitNow controls
// whether the change is committed immediately (client) or left for an explicit
// transaction commit (Txn).
func (d *Dgo) execMutate(ctx context.Context, runner txnRunner, commitNow bool, data any, deep ...bool) error {
	query, mu, err := ParseMutation(data, deep...)
	if err != nil {
		return fmt.Errorf("dgo: build mutation: %w", err)
	}
	if d.Debug {
		d.debugMutation(query, mu[0])
	}
	req := &api.Request{
		Query:     query,
		Mutations: mu,
		CommitNow: commitNow,
	}
	resp, err := runner.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("dgo: mutate: %w", err)
	}
	if d.Debug {
		d.logf("resp Uids: %+v\n", resp.Uids)
	}
	// For an insert (no uid yet) the conditional mutation must create the root node,
	// so its blank node appears in resp.Uids; its absence means the unique condition
	// matched an existing node and nothing was written. For an update (uid already
	// set) the mutation addresses <uid> directly and creates no new UID, so success
	// is signalled solely by the absence of a DGraph error.
	if rootUID(data) == "" {
		blankNode, err := BlankNodeName(data)
		if err != nil {
			return fmt.Errorf("dgo: inject node name: %w", err)
		}
		if _, ok := resp.Uids[blankNode]; !ok {
			return ErrDuplicate
		}
	}
	// resp.Uids still carries any newly created nested/blank nodes; distribute them.
	return SetUIDs(data, resp.Uids)
}

// execUpdate builds and runs an update on the given runner.
func (d *Dgo) execUpdate(ctx context.Context, runner txnRunner, commitNow bool, data any, fields ...string) error {
	query, mu, err := ParseUpdate(data, fields...)
	if err != nil {
		return fmt.Errorf("dgo: build mutation: %w", err)
	}
	if d.Debug {
		d.debugMutation(query, mu[0])
	}
	req := &api.Request{
		Query:     query,
		Mutations: mu,
		CommitNow: commitNow,
	}
	resp, err := runner.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("dgo: mutate: %w", err)
	}
	if d.Debug {
		d.logf("resp Uids: %+v\n", resp.Uids)
	}
	// ParseUpdate emits blank nodes (e.g. _:staffs0) for slice elements that have no
	// uid yet; DGraph returns their new UIDs keyed by predicate+index. Write them back
	// into the struct so callers see the created nodes' UIDs.
	if len(resp.Uids) > 0 {
		return SetUIDs(data, resp.Uids)
	}
	return nil
}

func (d *Dgo) Mutate(ctx context.Context, data any, deep ...bool) error {
	return d.execMutate(ctx, d.DG.NewTxn(), true, data, deep...)
}

func (d *Dgo) Update(ctx context.Context, data any, fields ...string) error {
	return d.execUpdate(ctx, d.DG.NewTxn(), true, data, fields...)
}

// Query runs a raw DQL query and returns the JSON response bytes.
func (d *Dgo) Query(ctx context.Context, dql string) ([]byte, error) {
	resp, err := d.DG.NewTxn().Query(ctx, dql)
	if err != nil {
		return nil, fmt.Errorf("dgo: query: %w", err)
	}
	return resp.Json, nil
}

// Txn represents an open Dgraph transaction. Use NewTxn to create one.
// Call Discard (typically via defer) to release resources, and Commit to persist.
// If any operation returns an error, call Discard to roll back.
type Txn struct {
	d   *Dgo
	txn interface {
		Do(ctx context.Context, req *api.Request) (*api.Response, error)
		Commit(ctx context.Context) error
		Discard(ctx context.Context) error
	}
}

// NewTxn opens a new read-write transaction.
// Typical usage:
//
//	txn := d.NewTxn()
//	defer txn.Discard(ctx)
//	if err := txn.Mutate(ctx, &user); err != nil { return err }
//	return txn.Commit(ctx)
func (d *Dgo) NewTxn() *Txn {
	return &Txn{d: d, txn: d.DG.NewTxn()}
}

// Commit commits the transaction. Returns an error if the commit fails.
func (t *Txn) Commit(ctx context.Context) error {
	return t.txn.Commit(ctx)
}

// Discard releases the transaction resources. Safe to call after Commit.
// Should be called via defer to ensure cleanup on error paths.
func (t *Txn) Discard(ctx context.Context) {
	_ = t.txn.Discard(ctx)
}

// DoTxn runs fn inside a single transaction. If fn returns an error the
// transaction is discarded (rolled back); otherwise it is committed.
func (d *Dgo) DoTxn(ctx context.Context, fn func(txn *Txn) error) error {
	txn := d.NewTxn()
	if err := fn(txn); err != nil {
		txn.Discard(ctx)
		return err
	}
	return txn.Commit(ctx)
}

// Mutate executes a mutation within the transaction without committing.
func (t *Txn) Mutate(ctx context.Context, data any, deep ...bool) error {
	return t.d.execMutate(ctx, t.txn, false, data, deep...)
}

// Update executes an update within the transaction without committing.
func (t *Txn) Update(ctx context.Context, data any, fields ...string) error {
	return t.d.execUpdate(ctx, t.txn, false, data, fields...)
}

type Query[T any] struct {
	d *Dgo
}

func Model[T any](d *Dgo) Query[T] {
	return Query[T]{d: d}
}

// First runs the filter and returns the first matching node. It forces first: 1
// on the query — any First/Offset set on the filter is overridden — so at most one
// node is fetched. Returns ErrNotFound (wrapped) when nothing matches.
func (q Query[T]) First(ctx context.Context, filter DgFilter) (*T, error) {
	// Only the first node is used, so cap the query at first: 1 to avoid fetching
	// a large result set when the filter matches many nodes.
	query := filter.DQuely().First(1).Query()
	resp, err := q.d.DG.NewTxn().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dgo: query: %w", err)
	}
	return q.parseData(resp.Json, filter.DgraphKey())
}

func (q Query[T]) Find(ctx context.Context, filter DgFilter) ([]T, error) {
	var result []T
	var query = filter.DQuely().Query()
	resp, err := q.d.DG.NewTxn().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dgo: query: %w", err)
	}
	result, _, err = q.parseDataMulti(resp.Json, filter.DgraphKey())
	return result, err
}

func (q Query[T]) FindAndCount(ctx context.Context, filter DgFilter) ([]T, int64, error) {
	var query = filter.DQuely().QueryAndCount()
	resp, err := q.d.DG.NewTxn().Query(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("dgo: query: %w", err)
	}
	return q.parseDataMulti(resp.Json, filter.DgraphKey())
}

func (q Query[T]) parseDataMulti(data []byte, key string) ([]T, int64, error) {
	var raw map[string]json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, 0, fmt.Errorf("dquely: decode response: %w", err)
	}

	block, ok := raw[key]
	if !ok {
		return nil, 0, fmt.Errorf("dquely: block %q not found in response (check the DgraphKey)", key)
	}

	var arr []T
	type count struct {
		Total int64 `json:"total"`
	}
	var total int64
	var counter []count

	if err := json.Unmarshal(block, &arr); err != nil {
		return nil, 0, fmt.Errorf("dquely: decode block %q: %w", key, err)
	}

	if err := json.Unmarshal(raw["total"], &counter); err == nil {
		if len(counter) > 0 {
			total = counter[0].Total
		}
	}

	return arr, total, nil
}

func (q Query[T]) parseData(data []byte, key string) (*T, error) {
	arr, _, err := q.parseDataMulti(data, key)
	if err != nil {
		return nil, err
	}

	if len(arr) == 0 {
		return nil, ErrNotFound
	}

	return &arr[0], nil
}
