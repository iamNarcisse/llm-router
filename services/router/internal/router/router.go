package router

import (
	"context"
	"fmt"
	"time"

	"github.com/qdrant/go-client/qdrant"

	config "llm-router/services/router/internal/config"
	embedding "llm-router/services/router/internal/embedding"
)

type RouteResult struct {
	Route          string
	Model          string
	Confidence     float32
	TotalLatencyMs float32
	EmbeddingMs    float32
	VectorSearchMs float32
	Metadata       map[string]string
}

type Router struct {
	embedder     *embedding.Client
	qdrantClient *qdrant.Client
	collection   string
	config       *config.RoutingConfig
}

func New(embedder *embedding.Client, qdrantCfg *config.QdrantConfig, routingCfg *config.RoutingConfig) *Router {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: qdrantCfg.Host,
		Port: qdrantCfg.Port,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create qdrant client: %v", err))
	}

	return &Router{
		embedder:     embedder,
		qdrantClient: client,
		collection:   qdrantCfg.Collection,
		config:       routingCfg,
	}
}

func (r *Router) Route(ctx context.Context, query string, topK int, threshold float32, filters map[string]string) (*RouteResult, error) {
	start := time.Now()
	result := &RouteResult{
		Metadata: make(map[string]string),
	}

	// Apply defaults
	if topK == 0 {
		topK = r.config.TopK
	}
	if threshold == 0 {
		threshold = r.config.ScoreThreshold
	}

	// Get embedding
	embedResult, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}
	result.EmbeddingMs = embedResult.LatencyMs

	// Build filter
	var filter *qdrant.Filter
	if len(filters) > 0 {
		conditions := make([]*qdrant.Condition, 0, len(filters))
		for key, value := range filters {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: key,
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: value,
							},
						},
					},
				},
			})
		}
		filter = &qdrant.Filter{Must: conditions}
	}

	// Search Qdrant
	searchStart := time.Now()
	results, err := r.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: r.collection,
		Query:          qdrant.NewQuery(embedResult.Vector...),
		Limit:          qdrant.PtrOf(uint64(topK)),
		Filter:         filter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}
	result.VectorSearchMs = float32(time.Since(searchStart).Milliseconds())

	// Select best route
	route := r.selectRoute(results, threshold)
	result.Route = route.Name
	result.Model = route.Model
	result.Confidence = route.Score
	result.Metadata = route.Metadata
	result.TotalLatencyMs = float32(time.Since(start).Milliseconds())

	return result, nil
}

type routeCandidate struct {
	Name     string
	Model    string
	Score    float32
	Metadata map[string]string
}

func (r *Router) selectRoute(matches []*qdrant.ScoredPoint, threshold float32) routeCandidate {
	if len(matches) == 0 || matches[0].Score < threshold {
		return routeCandidate{
			Name:     "default",
			Model:    r.config.DefaultModel,
			Score:    0,
			Metadata: make(map[string]string),
		}
	}

	// Aggregate scores by route
	scoresByRoute := make(map[string]float32)
	metaByRoute := make(map[string]map[string]string)
	modelByRoute := make(map[string]string)

	for _, m := range matches {
		payload := m.Payload
		if payload == nil {
			continue
		}

		routeName := payload["route"].GetStringValue()
		scoresByRoute[routeName] += m.Score

		if _, exists := metaByRoute[routeName]; !exists {
			metaByRoute[routeName] = make(map[string]string)
			modelByRoute[routeName] = payload["model"].GetStringValue()

			// Extract metadata
			for key, val := range payload {
				if key != "route" && key != "model" && key != "utterance" {
					metaByRoute[routeName][key] = val.GetStringValue()
				}
			}
		}
	}

	// Find best
	var best routeCandidate
	for route, score := range scoresByRoute {
		if score > best.Score {
			best = routeCandidate{
				Name:     route,
				Model:    modelByRoute[route],
				Score:    score,
				Metadata: metaByRoute[route],
			}
		}
	}

	return best
}

func (r *Router) HealthCheck(ctx context.Context) (embeddingOK bool, qdrantOK bool) {
	if err := r.embedder.HealthCheck(ctx); err == nil {
		embeddingOK = true
	}

	if _, err := r.qdrantClient.HealthCheck(ctx); err == nil {
		qdrantOK = true
	}

	return
}

func (r *Router) Close() error {
	return r.qdrantClient.Close()
}
