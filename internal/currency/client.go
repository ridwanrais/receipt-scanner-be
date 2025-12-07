package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	frankfurterBaseURL = "https://api.frankfurter.dev/v1"
	cacheTTL           = 1 * time.Hour
)

// ExchangeRates represents the response from Frankfurter API
type ExchangeRates struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// Client handles currency conversion using Frankfurter API
type Client struct {
	httpClient *http.Client
	cache      map[string]*cachedRates
	cacheMu    sync.RWMutex
}

type cachedRates struct {
	rates     *ExchangeRates
	expiresAt time.Time
}

// NewClient creates a new currency client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]*cachedRates),
	}
}

// GetLatestRates fetches the latest exchange rates for a base currency
func (c *Client) GetLatestRates(ctx context.Context, baseCurrency string) (*ExchangeRates, error) {
	cacheKey := fmt.Sprintf("latest_%s", baseCurrency)

	// Check cache
	c.cacheMu.RLock()
	if cached, ok := c.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		c.cacheMu.RUnlock()
		return cached.rates, nil
	}
	c.cacheMu.RUnlock()

	// Fetch from API
	url := fmt.Sprintf("%s/latest?base=%s", frankfurterBaseURL, baseCurrency)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var rates ExchangeRates
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the result
	c.cacheMu.Lock()
	c.cache[cacheKey] = &cachedRates{
		rates:     &rates,
		expiresAt: time.Now().Add(cacheTTL),
	}
	c.cacheMu.Unlock()

	return &rates, nil
}

// Convert converts an amount from one currency to another
func (c *Client) Convert(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}

	// Get rates with fromCurrency as base
	rates, err := c.GetLatestRates(ctx, fromCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get exchange rates: %w", err)
	}

	rate, ok := rates.Rates[toCurrency]
	if !ok {
		return 0, fmt.Errorf("exchange rate not found for %s to %s", fromCurrency, toCurrency)
	}

	return amount * rate, nil
}

// GetSupportedCurrencies returns a list of supported currencies
func (c *Client) GetSupportedCurrencies(ctx context.Context) ([]string, error) {
	rates, err := c.GetLatestRates(ctx, "EUR")
	if err != nil {
		return nil, err
	}

	currencies := make([]string, 0, len(rates.Rates)+1)
	currencies = append(currencies, "EUR") // Add base currency
	for currency := range rates.Rates {
		currencies = append(currencies, currency)
	}

	return currencies, nil
}
