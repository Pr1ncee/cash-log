package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

// PreciousMetalsApi represents the precious metals api
type PreciousMetalsApi struct {
	ApiUsingConfig
	preciousMetals *services.PreciousMetalsService
}

// Initialize a precious metals api singleton instance
var (
	PreciousMetalsHandler = &PreciousMetalsApi{
		ApiUsingConfig: ApiUsingConfig{
			container: settings.Container,
		},
		preciousMetals: services.PreciousMetals,
	}
)

// PriceHistoryHandler returns price history for a precious metal
func (a *PreciousMetalsApi) PriceHistoryHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.PreciousMetalPriceRequest
	err := c.ShouldBindQuery(&req)

	if err != nil {
		log.Warnf(c, "[precious_metals.PriceHistoryHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	result, apiErr := a.preciousMetals.GetPriceHistory(c, req.Metal, req.Currency, req.Unit, req.Timeline)
	if apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

// PortfolioHandler returns the user's precious metals portfolio value
func (a *PreciousMetalsApi) PortfolioHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.PreciousMetalPortfolioRequest
	err := c.ShouldBindQuery(&req)

	if err != nil {
		log.Warnf(c, "[precious_metals.PortfolioHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	// For now, return the current price - portfolio calculation will be implemented in Phase 5
	if req.Metal != "" {
		result, apiErr := a.preciousMetals.GetCurrentPrice(c, req.Metal, "USD", "grams")
		if apiErr != nil {
			return nil, apiErr
		}
		return result, nil
	}

	return &models.PreciousMetalPortfolioResponse{
		Items:      []*models.PreciousMetalPortfolioItem{},
		TotalValue: 0,
		TotalCost:  0,
	}, nil
}

// RefreshHandler triggers a price refresh from the GoldAPI service
func (a *PreciousMetalsApi) RefreshHandler(c *core.WebContext) (any, *errs.Error) {
	var req models.PreciousMetalRefreshRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		log.Warnf(c, "[precious_metals.RefreshHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	result, apiErr := a.preciousMetals.RefreshPrice(c, req.Metal, req.Currency)
	if apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}
