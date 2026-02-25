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
	DG *dgo.Dgraph
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

func (d *Dgo) Mutate(ctx context.Context, data any, deep ...bool) error {
	query, mu, err := ParseMutation(data, deep...)
	if err != nil {
		return fmt.Errorf("dgo: build mutation: %w", err)
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

type Query[T any] struct {
	d *Dgo
}

func Model[T any](d *Dgo) Query[T] {
	return Query[T]{d: d}
}

func (q Query[T]) First(ctx context.Context, filter DgFilter) (*T, error) {
	var result *T
	var query = filter.Query()
	resp, err := q.d.DG.RunDQL(context.TODO(), query)
	if err != nil {
		return nil, fmt.Errorf("dgo: query: %w", err)
	}
	result, err = q.parseData(resp.Json, filter.DgraphKey())
	return result, nil
}

func (q Query[T]) parseData(data []byte, key string) (*T, error) {
	var raw map[string]json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var arr []T
	if err := json.Unmarshal(raw[key], &arr); err != nil {
		return nil, err
	}

	if len(arr) == 0 {
		return nil, errors.New("empty result")
	}

	return &arr[0], nil
}
