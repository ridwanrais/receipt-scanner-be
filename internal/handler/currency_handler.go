package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ridwanfathin/invoice-processor-service/internal/currency"
)

// CurrencyHandler handles currency-related endpoints
type CurrencyHandler struct {
	currencyClient *currency.Client
}

// NewCurrencyHandler creates a new currency handler
func NewCurrencyHandler(client *currency.Client) *CurrencyHandler {
	return &CurrencyHandler{
		currencyClient: client,
	}
}

// GetExchangeRates returns exchange rates for a base currency
// @Summary Get exchange rates
// @Description Get latest exchange rates for a base currency
// @Tags currency
// @Accept json
// @Produce json
// @Param base query string false "Base currency (default: USD)"
// @Success 200 {object} currency.ExchangeRates "Exchange rates"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/currency/rates [get]
func (h *CurrencyHandler) GetExchangeRates(c *gin.Context) {
	baseCurrency := c.DefaultQuery("base", "USD")

	rates, err := h.currencyClient.GetLatestRates(c.Request.Context(), baseCurrency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": "Failed to fetch exchange rates: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, rates)
}

// ConvertCurrency converts an amount from one currency to another
// @Summary Convert currency
// @Description Convert an amount from one currency to another
// @Tags currency
// @Accept json
// @Produce json
// @Param amount query number true "Amount to convert"
// @Param from query string true "Source currency"
// @Param to query string true "Target currency"
// @Success 200 {object} map[string]interface{} "Conversion result"
// @Failure 400 {object} model.ErrorResponse "Bad request"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/currency/convert [get]
func (h *CurrencyHandler) ConvertCurrency(c *gin.Context) {
	amountStr := c.Query("amount")
	fromCurrency := c.Query("from")
	toCurrency := c.Query("to")

	if amountStr == "" || fromCurrency == "" || toCurrency == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "amount, from, and to parameters are required",
		})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "400",
			"message": "Invalid amount",
		})
		return
	}

	convertedAmount, err := h.currencyClient.Convert(c.Request.Context(), amount, fromCurrency, toCurrency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": "Failed to convert currency: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"amount":          amount,
		"from":            fromCurrency,
		"to":              toCurrency,
		"convertedAmount": convertedAmount,
	})
}

// GetSupportedCurrencies returns a list of supported currencies
// @Summary Get supported currencies
// @Description Get list of all supported currencies
// @Tags currency
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "List of currencies"
// @Failure 500 {object} model.ErrorResponse "Internal server error"
// @Router /v1/currency/supported [get]
func (h *CurrencyHandler) GetSupportedCurrencies(c *gin.Context) {
	currencies, err := h.currencyClient.GetSupportedCurrencies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "500",
			"message": "Failed to fetch supported currencies: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currencies": currencies,
	})
}

// RegisterCurrencyRoutes registers currency routes
func (h *CurrencyHandler) RegisterCurrencyRoutes(router *gin.RouterGroup) {
	currencyGroup := router.Group("/currency")
	{
		currencyGroup.GET("/rates", h.GetExchangeRates)
		currencyGroup.GET("/convert", h.ConvertCurrency)
		currencyGroup.GET("/supported", h.GetSupportedCurrencies)
	}
}
