// Copyright 2022 Azugo. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/goccy/go-json"
)

type redisCache[T any] struct {
	con    *redis.Client
	prefix string
	ttl    time.Duration
	loader func(ctx context.Context, key string) (interface{}, error)
}

func newRedisCache[T any](prefix string, con *redis.Client, opts ...CacheOption) (CacheInstance[T], error) {
	opt := newCacheOptions(opts...)

	keyPrefix := opt.KeyPrefix
	if keyPrefix != "" {
		keyPrefix += ":"
	}

	return &redisCache[T]{
		con:    con,
		prefix: keyPrefix + prefix + ":",
		ttl:    opt.TTL,
		loader: opt.Loader,
	}, nil
}

func newRedisClient(constr, password string) (*redis.Client, error) {
	redisOptions, err := redis.ParseURL(constr)
	if err != nil {
		return nil, err
	}
	// If password is provided override provided in connection string.
	if len(password) != 0 {
		redisOptions.Password = password
	}

	return redis.NewClient(redisOptions), nil
}

func (c *redisCache[T]) Get(ctx context.Context, key string, opts ...ItemOption[T]) (T, error) {
	val := new(T)
	if c.con == nil {
		return *val, ErrCacheClosed
	}
	s := c.con.Get(ctx, c.prefix+key)
	if s.Err() == redis.Nil {
		if c.loader != nil {
			v, err := c.loader(ctx, key)
			if err != nil {
				return *val, err
			}
			vv, ok := v.(T)
			if !ok {
				return *val, fmt.Errorf("invalid value from loader: %v", v)
			}
			if err := c.Set(ctx, key, vv, opts...); err != nil {
				return *val, err
			}
			return vv, nil
		}
		return *val, nil
	}
	if s.Err() != nil {
		return *val, s.Err()
	}
	if err := json.Unmarshal([]byte(s.Val()), val); err != nil {
		return *val, fmt.Errorf("invalid cache value: %w", err)
	}
	return *val, nil
}

func (c *redisCache[T]) Pop(ctx context.Context, key string) (T, error) {
	val := new(T)
	if c.con == nil {
		return *val, ErrCacheClosed
	}
	s := c.con.GetDel(ctx, c.prefix+key)
	if s.Err() == redis.Nil {
		return *val, ErrKeyNotFound{Key: key}
	}
	if s.Err() != nil {
		return *val, s.Err()
	}
	if err := json.Unmarshal([]byte(s.Val()), val); err != nil {
		return *val, fmt.Errorf("invalid cache value: %w", err)
	}
	return *val, nil
}

func (c *redisCache[T]) Set(ctx context.Context, key string, value T, opts ...ItemOption[T]) error {
	if c.con == nil {
		return ErrCacheClosed
	}
	buf, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("invalid cache value: %w", err)
	}
	opt := newItemOptions(opts...)
	ttl := c.ttl
	if opt.TTL != 0 {
		ttl = opt.TTL
	}
	s := c.con.Set(ctx, c.prefix+key, string(buf), ttl)
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}

func (c *redisCache[T]) Delete(ctx context.Context, key string) error {
	if c.con == nil {
		return ErrCacheClosed
	}
	s := c.con.Del(ctx, c.prefix+key)
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}

func (c *redisCache[T]) Ping(ctx context.Context) error {
	if c.con == nil {
		return nil
	}
	s := c.con.Ping(ctx)
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}

func (c *redisCache[T]) Close() {
	if c.con == nil {
		return
	}
	_ = c.con.Close()
	c.con = nil
}
