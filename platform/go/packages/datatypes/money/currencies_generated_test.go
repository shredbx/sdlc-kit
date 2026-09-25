package money

import "testing"

// TestGeneratedCurrenciesRegistered proves the 8 governed currencies from
// currencies.yml (projected into currencies.generated.go) are registered by
// init() with the correct decimals, symbol, and symbol position. This is the
// behavioral contract that the generated table feeds the runtime registry.
func TestGeneratedCurrenciesRegistered(t *testing.T) {
	want := []Currency{
		{Code: "THB", Symbol: "฿", Name: "Thai Baht", Decimals: 2, SymbolPosition: "prefix"},
		{Code: "USD", Symbol: "$", Name: "US Dollar", Decimals: 2, SymbolPosition: "prefix"},
		{Code: "EUR", Symbol: "€", Name: "Euro", Decimals: 2, SymbolPosition: "suffix"},
		{Code: "GBP", Symbol: "£", Name: "British Pound", Decimals: 2, SymbolPosition: "prefix"},
		{Code: "JPY", Symbol: "¥", Name: "Japanese Yen", Decimals: 0, SymbolPosition: "prefix"},
		{Code: "CNY", Symbol: "¥", Name: "Chinese Yuan", Decimals: 2, SymbolPosition: "prefix"},
		{Code: "RUB", Symbol: "₽", Name: "Russian Ruble", Decimals: 2, SymbolPosition: "suffix"},
		{Code: "AUD", Symbol: "A$", Name: "Australian Dollar", Decimals: 2, SymbolPosition: "prefix"},
	}

	for _, w := range want {
		got, ok := GetCurrency(w.Code)
		if !ok {
			t.Errorf("%s not registered", w.Code)
			continue
		}
		if got.Symbol != w.Symbol {
			t.Errorf("%s symbol = %q, want %q", w.Code, got.Symbol, w.Symbol)
		}
		if got.Name != w.Name {
			t.Errorf("%s name = %q, want %q", w.Code, got.Name, w.Name)
		}
		if got.Decimals != w.Decimals {
			t.Errorf("%s decimals = %d, want %d", w.Code, got.Decimals, w.Decimals)
		}
		if got.SymbolPosition != w.SymbolPosition {
			t.Errorf("%s symbol_position = %q, want %q", w.Code, got.SymbolPosition, w.SymbolPosition)
		}
	}
}

// TestGeneratedCurrenciesTableLength guards against silent drops/dupes in the
// generated table.
func TestGeneratedCurrenciesTableLength(t *testing.T) {
	if len(generatedCurrencies) != 8 {
		t.Errorf("generatedCurrencies has %d entries, want 8", len(generatedCurrencies))
	}
}
