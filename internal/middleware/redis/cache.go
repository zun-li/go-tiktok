package redis

import (
	"context"
	"errors"
	"time"
)

func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("redis client not initialized")
	}
	return c.rdb.Get(ctx, key).Bytes()
}

func (c *Client) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Del(ctx context.Context, key string) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Del(ctx, key).Err()
}

func (c *Client) DelByPattern(ctx context.Context, pattern string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	iter := c.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		_ = c.rdb.Del(ctx, iter.Val())
	}
	return iter.Err()
}

func (c *Client) MGet(cacheCtx context.Context, cacheKeys ...string) ([]interface{}, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("redis client not initialized")
	}
	return c.rdb.MGet(cacheCtx, cacheKeys...).Result()
}

// MarkChunkUploaded atomically updates one bit without overwriting concurrent uploads.
func (c *Client) MarkChunkUploaded(ctx context.Context, key string, index int, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	const script = `
local raw = redis.call('GET', KEYS[1])
if not raw then return 0 end
local session = cjson.decode(raw)
session.uploaded_bits[tonumber(ARGV[1]) + 1] = true
redis.call('SET', KEYS[1], cjson.encode(session), 'PX', ARGV[2])
return 1
`
	result, err := c.rdb.Eval(ctx, script, []string{key}, index, ttl.Milliseconds()).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return errors.New("upload session expired")
	}
	return nil
}
