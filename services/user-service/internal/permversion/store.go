package permversion

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Store tracks per-user permission versions for JWT invalidation.
type Store interface {
	Current(ctx context.Context, userID uuid.UUID) (uint64, error)
	BumpUsers(ctx context.Context, userIDs []uuid.UUID) error
}

type Noop struct{}

func (Noop) Current(context.Context, uuid.UUID) (uint64, error) { return 1, nil }

func (Noop) BumpUsers(context.Context, []uuid.UUID) error { return nil }

type Redis struct {
	client redis.UniversalClient
}

func NewRedis(url string) (*Redis, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Redis{client: client}, nil
}

func key(userID uuid.UUID) string {
	return fmt.Sprintf("perm:ver:%s", userID.String())
}

func (r *Redis) Current(ctx context.Context, userID uuid.UUID) (uint64, error) {
	k := key(userID)
	val, err := r.client.Get(ctx, k).Uint64()
	if err == redis.Nil {
		if err := r.client.Set(ctx, k, 1, 0).Err(); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	if val == 0 {
		return 1, nil
	}
	return val, nil
}

func (r *Redis) BumpUsers(ctx context.Context, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	pipe := r.client.Pipeline()
	for _, id := range userIDs {
		pipe.Incr(ctx, key(id))
	}
	_, err := pipe.Exec(ctx)
	return err
}
