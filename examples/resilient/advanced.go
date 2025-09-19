package resilient

import (
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

// HighlyResilientService combines tracing and multiple resilience patterns
type HighlyResilientService struct {
	auth   *gauth.GAuth
	server *gauth.ResourceServer
	// composite *resilience.Composite // TODO: Implement or remove composite pattern
	// tracer    // TODO: Implement or remove tracing
}

func NewHighlyResilientService(auth *gauth.GAuth) (*HighlyResilientService, error) {
       // TODO: Implement tracing and composite pattern if needed
       return &HighlyResilientService{
	       auth:   auth,
	       server: gauth.NewResourceServer("resilient-service", auth),
       }, nil
}

// func (s *HighlyResilientService) ProcessRequest(ctx context.Context, tx gauth.TransactionDetails, token string) error {
//      // TODO: Implement tracing and composite pattern if needed
//      return nil
// }
