package errs

import "net/http"

// Sub category for precious metals errors
const NormalSubcategoryPreciousMetals = 20

// Precious metals error codes
var (
	ErrPreciousMetalsNotEnabled       = NewNormalError(NormalSubcategoryPreciousMetals, 0, http.StatusBadRequest, "precious metals feature is not enabled")
	ErrPreciousMetalsDataNotFound     = NewNormalError(NormalSubcategoryPreciousMetals, 1, http.StatusNotFound, "precious metals data not found")
	ErrPreciousMetalsAPIRateLimited   = NewNormalError(NormalSubcategoryPreciousMetals, 2, http.StatusTooManyRequests, "precious metals api can only be called once per 24 hours")
	ErrPreciousMetalsAPIKeyMissing    = NewNormalError(NormalSubcategoryPreciousMetals, 3, http.StatusBadRequest, "precious metals api key is not configured")
	ErrPreciousMetalsAPIRequestFailed = NewNormalError(NormalSubcategoryPreciousMetals, 4, http.StatusBadGateway, "precious metals api request failed")
)
