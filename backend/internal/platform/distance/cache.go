package distance

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

const (
	cacheSize = 50_000
	cacheTTL  = 24 * time.Hour
)

type CachedProvider struct {
	inner Provider
	cache *expirable.LRU[string, Edge]
}

func NewCachedProvider(inner Provider) *CachedProvider {
	return &CachedProvider{
		inner: inner,
		cache: expirable.NewLRU[string, Edge](cacheSize, nil, cacheTTL),
	}
}

func pairKey(a, b Point) string {
	return fmt.Sprintf("%.6f,%.6f->%.6f,%.6f", a.Lat, a.Lng, b.Lat, b.Lng)
}

func (c *CachedProvider) GetMatrix(ctx context.Context, points []Point) ([][]Edge, error) {
	n := len(points)
	if n == 0 {
		return nil, nil
	}

	matrix := make([][]Edge, n)
	for i := range matrix {
		matrix[i] = make([]Edge, n)
	}

	missing := false
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if e, ok := c.cache.Get(pairKey(points[i], points[j])); ok {
				matrix[i][j] = e
			} else {
				missing = true
			}
		}
	}

	if !missing {
		return matrix, nil
	}

	fresh, err := c.inner.GetMatrix(ctx, points)
	if err != nil {
		return nil, err
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			matrix[i][j] = fresh[i][j]
			c.cache.Add(pairKey(points[i], points[j]), fresh[i][j])
		}
	}

	return matrix, nil
}
