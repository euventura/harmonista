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
		log.Println("WARNING: Valkey client is nil, analytics will be disabled")
		return nil
	}

	log.Println("SUCCESS: Analytics module initialized with Valkey client")
	return &AnalyticsModule{vk: vk}
}

// key builds the Valkey key: analytics.{blog}.{page}
func key(blog, page string) string {
	return "analytics." + blog + "." + page
}

// TrackVisit incrementa o contador de visitas para uma página
func (a *AnalyticsModule) TrackVisit(blog string, page string) {
	if a == nil || a.vk == nil {
		log.Printf("DEBUG: TrackVisit called with nil module for blog=%s, page=%s", blog, page)
		return
	}

	log.Printf("DEBUG: Tracking visit for blog=%s, page=%s, key=%s", blog, page, key(blog, page))
	go func() {
		if err := a.vk.Do(ctx, a.vk.B().Incr().Key(key(blog, page)).Build()).Error(); err != nil {
			log.Printf("ERROR: Failed to track visit for %s/%s: %v", blog, page, err)
		} else {
			log.Printf("DEBUG: Successfully tracked visit for %s/%s", blog, page)
		}
	}()
}

// GetPageVisits retorna o número de visitas de uma página específica
func (a *AnalyticsModule) GetPageVisits(blog string, page string) int64 {
	if a == nil || a.vk == nil {
		log.Printf("DEBUG: GetPageVisits called with nil module for blog=%s, page=%s", blog, page)
		return 0
	}

	log.Printf("DEBUG: Getting page visits for blog=%s, page=%s, key=%s", blog, page, key(blog, page))
	val, err := a.vk.Do(ctx, a.vk.B().Get().Key(key(blog, page)).Build()).AsInt64()
	if err != nil {
		log.Printf("ERROR: Failed to get page visits for %s/%s: %v", blog, page, err)
		return 0
	}
	log.Printf("DEBUG: Page visits for %s/%s: %d", blog, page, val)
	return val
}

// GetAllBlogVisits retorna mapa de página→contagem para um blog
func (a *AnalyticsModule) GetAllBlogVisits(blog string) map[string]int64 {
	if a == nil || a.vk == nil {
		return nil
	}

	result := make(map[string]int64)
	prefix := "analytics." + blog + "."

	log.Printf("DEBUG: Scanning all blog visits for blog=%s, prefix=%s", blog, prefix)
	var cursor uint64
	for {
		entry, err := a.vk.Do(ctx, a.vk.B().Scan().Cursor(cursor).Match(prefix+"*").Count(100).Build()).AsScanEntry()
		if err != nil {
			log.Printf("ERROR: Failed to scan blog visits for %s: %v", blog, err)
			break
		}
		log.Printf("DEBUG: Scan returned %d keys, cursor=%d", len(entry.Elements), entry.Cursor)
		for _, k := range entry.Elements {
			val, err := a.vk.Do(ctx, a.vk.B().Get().Key(k).Build()).AsInt64()
			if err != nil {
				log.Printf("WARNING: Failed to get value for key %s: %v", k, err)
				continue
			}
			page := strings.TrimPrefix(k, prefix)
			result[page] = val
		}
		cursor = entry.Cursor
		if cursor == 0 {
			log.Printf("DEBUG: Scan complete for blog=%s, total keys found=%d", blog, len(result))
			break
		}
	}

	return result
}

// GetTotalVisits retorna o total de visitas de um blog
func (a *AnalyticsModule) GetTotalVisits(blog string) int64 {
	all := a.GetAllBlogVisits(blog)
	var total int64
	for _, count := range all {
		total += count
	}
	return total
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
