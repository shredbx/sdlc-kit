package property

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/repository"
	"gopkg.in/yaml.v3"
)

var _ repository.Repository[Property] = (*YAMLStore)(nil)

type YAMLStore struct {
	mu    sync.RWMutex
	items map[string]Property
}

func NewYAMLStore(seedPath string) (*YAMLStore, error) {
	data, err := os.ReadFile(seedPath)
	if err != nil {
		return nil, fmt.Errorf("read seed file: %w", err)
	}

	var list []Property
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse seed YAML: %w", err)
	}

	items := make(map[string]Property, len(list))
	for _, p := range list {
		if p.ID == "" {
			p.ID = uuid.NewString()
		}
		items[p.ID] = p
	}

	return &YAMLStore{items: items}, nil
}

func (s *YAMLStore) Get(_ context.Context, id string) (Property, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.items[id]
	if !ok {
		return Property{}, repository.ErrNotFound
	}
	return p, nil
}

func (s *YAMLStore) List(_ context.Context, opts repository.ListOptions) ([]Property, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Property
	for _, p := range s.items {
		if p.DeletedAt != nil {
			continue
		}
		if !matchQuery(p, opts.Filter) {
			continue
		}
		result = append(result, p)
	}

	total := len(result)

	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		result = nil
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, total, nil
}

func (s *YAMLStore) Create(_ context.Context, p Property) (Property, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	s.items[p.ID] = p
	return p, nil
}

func (s *YAMLStore) Update(_ context.Context, id string, p Property) (Property, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return Property{}, repository.ErrNotFound
	}
	p.ID = id
	p.UpdatedAt = time.Now()
	s.items[id] = p
	return p, nil
}

func (s *YAMLStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.items[id]
	if !ok {
		return repository.ErrNotFound
	}
	now := time.Now()
	p.DeletedAt = &now
	s.items[id] = p
	return nil
}

func matchQuery(p Property, q repository.Query) bool {
	if q.IsEmpty() {
		return true
	}
	if q.Pred != nil {
		return matchPredicate(p, *q.Pred)
	}
	if len(q.And) > 0 {
		for _, sub := range q.And {
			if !matchQuery(p, sub) {
				return false
			}
		}
		return true
	}
	if len(q.Or) > 0 {
		for _, sub := range q.Or {
			if matchQuery(p, sub) {
				return true
			}
		}
		return false
	}
	return true
}

// matchPredicate evaluates one predicate against an in-memory Property so the
// YAML store mirrors what the Postgres builder filters on. It covers the ops +
// fields the catalog filter language actually emits (Eq on the identity/coded
// columns, Gte/Lte on prices + bedrooms, In on property_type — extended for the
// admin filter, RP-2/2607-004). Anything it doesn't model deliberately matches
// (pass-through) so an unrelated predicate never empties a test fixture — but
// keep this in step with the filter builders, or handler tests go fake-green.
func matchPredicate(p Property, pred repository.Predicate) bool {
	switch pred.Op {
	case repository.OpEq:
		switch pred.FieldName {
		case "is_published":
			if v, ok := pred.Value.(bool); ok {
				return p.IsPublished == v
			}
		case "for_sale":
			if v, ok := pred.Value.(bool); ok {
				return p.ForSale == v
			}
		case "for_lease":
			if v, ok := pred.Value.(bool); ok {
				return p.ForLease == v
			}
		case "property_type":
			if v, ok := pred.Value.(string); ok && p.PropertyType != nil {
				return string(*p.PropertyType) == v
			}
		case "province":
			if v, ok := pred.Value.(string); ok {
				return p.Address.Province == v
			}
		case "city":
			if v, ok := pred.Value.(string); ok {
				return p.Address.City == v
			}
		case "sub_district":
			if v, ok := pred.Value.(string); ok {
				return p.Address.SubDistrict == v
			}
		case "furnished":
			if v, ok := pred.Value.(string); ok {
				return p.Furnished != nil && string(*p.Furnished) == v
			}
		}
	case repository.OpGte:
		switch pred.FieldName {
		case "sale_price":
			if v, ok := pred.Value.(int64); ok {
				return p.SalePrice != nil && *p.SalePrice >= v
			}
		case "lease_price":
			if v, ok := pred.Value.(int64); ok {
				return p.LeasePrice != nil && *p.LeasePrice >= v
			}
		case "bedrooms":
			if v, ok := pred.Value.(int); ok {
				return p.Rooms != nil && p.Rooms.Bedrooms != nil && *p.Rooms.Bedrooms >= v
			}
		}
	case repository.OpLte:
		switch pred.FieldName {
		case "sale_price":
			if v, ok := pred.Value.(int64); ok {
				return p.SalePrice != nil && *p.SalePrice <= v
			}
		case "lease_price":
			if v, ok := pred.Value.(int64); ok {
				return p.LeasePrice != nil && *p.LeasePrice <= v
			}
		case "bedrooms":
			if v, ok := pred.Value.(int); ok {
				return p.Rooms != nil && p.Rooms.Bedrooms != nil && *p.Rooms.Bedrooms <= v
			}
		}
	case repository.OpIn:
		if pred.FieldName == "property_type" {
			if vals, ok := pred.Value.([]string); ok {
				if p.PropertyType == nil {
					return false
				}
				for _, v := range vals {
					if string(*p.PropertyType) == v {
						return true
					}
				}
				return false
			}
		}
	}
	return true
}
