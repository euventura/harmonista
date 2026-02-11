package analytics

import (
	"context"
	"log"
	"sort"
	"strings"

	"github.com/valkey-io/valkey-go"
)

var ctx = context.Background()

// PageVisits representa visitas de uma página
type PageVisits struct {
	Page       string
	Count      int64
	Percentage float64
}

// AnalyticsModule gerencia o tracking de analytics via Valkey
type AnalyticsModule struct {
	vk valkey.Client
}

// NewAnalyticsModule cria uma nova instância do módulo de analytics
func NewAnalyticsModule(vk valkey.Client) *AnalyticsModule {
	if vk == nil {
		log.Println("Valkey is nil, analytics will be disabled")
		return nil
	}

	log.Println("Analytics module initialized successfully (Valkey)")
	return &AnalyticsModule{vk: vk}
}

// key builds the Valkey key: analytics.{blog}.{page}
func key(blog, page string) string {
	return "analytics." + blog + "." + page
}

// TrackVisit incrementa o contador de visitas para uma página
func (a *AnalyticsModule) TrackVisit(blog string, page string) {
	if a == nil || a.vk == nil {
		return
	}

	go func() {
		if err := a.vk.Do(ctx, a.vk.B().Incr().Key(key(blog, page)).Build()).Error(); err != nil {
			log.Printf("Error tracking visit for %s/%s: %v", blog, page, err)
		}
	}()
}

// GetPageVisits retorna o número de visitas de uma página específica
func (a *AnalyticsModule) GetPageVisits(blog string, page string) int64 {
	if a == nil || a.vk == nil {
		return 0
	}

	val, err := a.vk.Do(ctx, a.vk.B().Get().Key(key(blog, page)).Build()).ToInt64()
	if err != nil {
		return 0
	}
	return val
}

// GetAllBlogVisits retorna mapa de página→contagem para um blog
func (a *AnalyticsModule) GetAllBlogVisits(blog string) map[string]int64 {
	if a == nil || a.vk == nil {
		return nil
	}

	result := make(map[string]int64)
	prefix := "analytics." + blog + "."

	var cursor uint64
	for {
		entry, err := a.vk.Do(ctx, a.vk.B().Scan().Cursor(cursor).Match(prefix+"*").Count(100).Build()).AsScanEntry()
		if err != nil {
			break
		}
		for _, k := range entry.Elements {
			val, err := a.vk.Do(ctx, a.vk.B().Get().Key(k).Build()).ToInt64()
			if err != nil {
				continue
			}
			page := strings.TrimPrefix(k, prefix)
			result[page] = val
		}
		cursor = entry.Cursor
		if cursor == 0 {
			break
		}
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
