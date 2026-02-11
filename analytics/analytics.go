package analytics

import (
	"context"
	"log"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// PageVisits representa visitas de uma página
type PageVisits struct {
	Page       string
	Count      int64
	Percentage float64
}

// AnalyticsModule gerencia o tracking de analytics via Redis
type AnalyticsModule struct {
	rdb *redis.Client
}

// NewAnalyticsModule cria uma nova instância do módulo de analytics
func NewAnalyticsModule(rdb *redis.Client) *AnalyticsModule {
	if rdb == nil {
		log.Println("Redis is nil, analytics will be disabled")
		return nil
	}

	log.Println("Analytics module initialized successfully (Redis)")
	return &AnalyticsModule{rdb: rdb}
}

// key builds the Redis key: analytics.{blog}.{page}
func key(blog, page string) string {
	return "analytics." + blog + "." + page
}

// TrackVisit incrementa o contador de visitas para uma página
func (a *AnalyticsModule) TrackVisit(blog string, page string) {
	if a == nil || a.rdb == nil {
		return
	}

	go func() {
		if err := a.rdb.Incr(ctx, key(blog, page)).Err(); err != nil {
			log.Printf("Error tracking visit for %s/%s: %v", blog, page, err)
		}
	}()
}

// GetPageVisits retorna o número de visitas de uma página específica
func (a *AnalyticsModule) GetPageVisits(blog string, page string) int64 {
	if a == nil || a.rdb == nil {
		return 0
	}

	val, err := a.rdb.Get(ctx, key(blog, page)).Int64()
	if err != nil {
		return 0
	}
	return val
}

// GetAllBlogVisits retorna mapa de página→contagem para um blog
func (a *AnalyticsModule) GetAllBlogVisits(blog string) map[string]int64 {
	if a == nil || a.rdb == nil {
		return nil
	}

	result := make(map[string]int64)
	prefix := "analytics." + blog + "."

	iter := a.rdb.Scan(ctx, 0, prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		k := iter.Val()
		val, err := a.rdb.Get(ctx, k).Int64()
		if err != nil {
			continue
		}
		page := strings.TrimPrefix(k, prefix)
		result[page] = val
	}

	return result
}

// GetTopPages retorna as top N páginas mais visitadas de um blog
func (a *AnalyticsModule) GetTopPages(blog string, limit int) []PageVisits {
	all := a.GetAllBlogVisits(blog)
	if len(all) == 0 {
		return nil
	}

	pages := make([]PageVisits, 0, len(all))
	for page, count := range all {
		pages = append(pages, PageVisits{Page: page, Count: count})
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Count > pages[j].Count
	})

	if limit > 0 && len(pages) > limit {
		pages = pages[:limit]
	}

	// Calculate percentages
	if len(pages) > 0 {
		max := pages[0].Count
		if max > 0 {
			for i := range pages {
				pages[i].Percentage = (float64(pages[i].Count) / float64(max)) * 100
			}
		}
	}

	return pages
}
