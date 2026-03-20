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

type Config struct {
	DNS       string `mapstructure:"dns"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	Namespace uint64 `mapstructure:"namespace"`
}

type Dgo struct {
	DG    *dgo.Dgraph
	Debug bool
}

// NewClient creates a Dgraph client and verifies connectivity.
// Call Close() when the client is no longer needed.
func NewClient(cfg Config) (*Dgo, error) {
	opts := []dgo.ClientOption{
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	if cfg.Username == "" || cfg.Password == "" {
		return nil, errors.New("dgo: username and password are required")
	}
	opts = append(opts, dgo.WithACLCreds(cfg.Username, cfg.Password))

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
	fmt.Printf("query: %s\n", query)
	fmt.Printf("condition: %s\n", mu.Cond)
	fmt.Printf("SetNquads: %s\n", mu.SetNquads)
	fmt.Printf("DelNquads: %s\n", mu.DelNquads)
}

func (d *Dgo) Mutate(ctx context.Context, data any, deep ...bool) error {
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
		CommitNow: true,
	}
	resp, err := d.DG.NewTxn().Do(ctx, req)
	if err != nil {
		return fmt.Errorf("dgo: mutate: %w", err)
	}
	if d.Debug {
		fmt.Printf("resp Uids: %+v\n", resp.Uids)
	}
	// If the conditional mutation fired, resp.Uids contains the new UID keyed by the
	// blank-node name. Write it back into the struct's dquely:"uid" field.
	blankNode, err := BlankNodeName(data)
	if err != nil {
		return fmt.Errorf("dgo: inject node name: %w", err)
	}
	_, ok := resp.Uids[blankNode]
	if !ok {
		// if there isn't node name mean the main node name was not inserted
		// because duplicate condition was not matched
		return fmt.Errorf("mutate failed: duplicated")
	}
	return SetUIDs(data, resp.Uids)
}

func (d *Dgo) Update(ctx context.Context, data any, fields ...string) error {
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
		CommitNow: true,
	}
	resp, err := d.DG.NewTxn().Do(ctx, req)
	if err != nil {
		return fmt.Errorf("dgo: mutate: %w", err)
	}
	if d.Debug {
		fmt.Printf("resp Uids: %+v\n", resp.Uids)
	}
	return nil
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
	query, mu, err := ParseMutation(data, deep...)
	if err != nil {
		return fmt.Errorf("dgo: build mutation: %w", err)
	}
	if t.d.Debug {
		t.d.debugMutation(query, mu[0])
	}
	req := &api.Request{
		Query:     query,
		Mutations: mu,
		CommitNow: false,
	}
	resp, err := t.txn.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("dgo: mutate: %w", err)
	}
	if t.d.Debug {
		fmt.Printf("resp Uids: %+v\n", resp.Uids)
	}
	blankNode, err := BlankNodeName(data)
	if err != nil {
		return fmt.Errorf("dgo: inject node name: %w", err)
	}
	if _, ok := resp.Uids[blankNode]; !ok {
		return fmt.Errorf("mutate failed: duplicated")
	}
	return SetUIDs(data, resp.Uids)
}

// Update executes an update within the transaction without committing.
func (t *Txn) Update(ctx context.Context, data any, fields ...string) error {
	query, mu, err := ParseUpdate(data, fields...)
	if err != nil {
		return fmt.Errorf("dgo: build mutation: %w", err)
	}
	if t.d.Debug {
		t.d.debugMutation(query, mu[0])
	}
	req := &api.Request{
		Query:     query,
		Mutations: mu,
		CommitNow: false,
	}
	resp, err := t.txn.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("dgo: mutate: %w", err)
	}
	if t.d.Debug {
		fmt.Printf("resp Uids: %+v\n", resp.Uids)
	}
	return nil
}

type Query[T any] struct {
	d *Dgo
}

func Model[T any](d *Dgo) Query[T] {
	return Query[T]{d: d}
}

func (q Query[T]) First(ctx context.Context, filter DgFilter) (*T, error) {
	var result *T
	var query = filter.Query()
	resp, err := q.d.DG.NewTxn().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dgo: query: %w", err)
	}
	result, err = q.parseData(resp.Json, filter.DgraphKey())
	return result, nil
}

func (q Query[T]) Find(ctx context.Context, filter DgFilter) ([]T, error) {
	var result []T
	var query = filter.Query()
	resp, err := q.d.DG.NewTxn().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("dgo: query: %w", err)
	}
	result, err = q.parseDataMulti(resp.Json, filter.DgraphKey())
	return result, nil
}

func (q Query[T]) parseDataMulti(data []byte, key string) ([]T, error) {
	var raw map[string]json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var arr []T
	if err := json.Unmarshal(raw[key], &arr); err != nil {
		return nil, err
	}

	return arr, nil
}

func (q Query[T]) parseData(data []byte, key string) (*T, error) {
	arr, err := q.parseDataMulti(data, key)
	if err != nil {
		return nil, err
	}

	if len(arr) == 0 {
		return nil, errors.New("empty result")
	}

	return &arr[0], nil
}
