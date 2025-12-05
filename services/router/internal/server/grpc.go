package server

import (
	"context"

	router "llm-router/services/router/internal/router"
	pb "llm-router/services/router/pkg/pb"
)

type RouterServer struct {
	pb.UnimplementedRouterServiceServer
	router *router.Router
}

func NewRouterServer(r *router.Router) *RouterServer {
	return &RouterServer{router: r}
}

func (s *RouterServer) Route(ctx context.Context, req *pb.RouteRequest) (*pb.RouteResponse, error) {
	result, err := s.router.Route(
		ctx,
		req.Query,
		int(req.TopK),
		req.ScoreThreshold,
		req.Filters,
	)
	if err != nil {
		return nil, err
	}

	return &pb.RouteResponse{
		Route:          result.Route,
		Model:          result.Model,
		Confidence:     result.Confidence,
		TotalLatencyMs: result.TotalLatencyMs,
		LatencyBreakdown: &pb.LatencyBreakdown{
			EmbeddingMs:    result.EmbeddingMs,
			VectorSearchMs: result.VectorSearchMs,
		},
		Metadata: result.Metadata,
	}, nil
}

func (s *RouterServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	embeddingOK, qdrantOK := s.router.HealthCheck(ctx)

	embeddingStatus := "unhealthy"
	if embeddingOK {
		embeddingStatus = "healthy"
	}

	qdrantStatus := "unhealthy"
	if qdrantOK {
		qdrantStatus = "healthy"
	}

	return &pb.HealthResponse{
		Healthy:                embeddingOK && qdrantOK,
		EmbeddingServiceStatus: embeddingStatus,
		QdrantStatus:           qdrantStatus,
	}, nil
}
