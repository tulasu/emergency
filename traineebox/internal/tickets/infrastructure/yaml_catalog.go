package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"traineebox/internal/tickets/domain/models"

	"gopkg.in/yaml.v3"
)

type yamlCatalog struct {
	loaded   []loadedType
	types    []models.IncidentType
	typeBy   map[string]*loadedType
	services []models.EmergencyService
	svcBy    map[string]models.EmergencyService
	routing  []routingRule
	routeBy  map[string]*routingRule
}

type loadedType struct {
	models.IncidentType
	Groups []models.IncidentTagGroup
}

type whenRule struct {
	TagsAny []string
	Add     []string
}

type routingRule struct {
	TypeCode string
	Always   []string
	When     []whenRule
}

type yamlTag struct {
	Code  string `yaml:"code"`
	Title string `yaml:"title"`
}

type yamlGroup struct {
	Code          string    `yaml:"code"`
	Title         string    `yaml:"title"`
	SelectionMode string    `yaml:"selection_mode"`
	ParentTag     string    `yaml:"parent_tag,omitempty"`
	Tags          []yamlTag `yaml:"tags"`
}

type yamlTypeFile struct {
	Code          string      `yaml:"code"`
	Title         string      `yaml:"title"`
	IncludeCommon []string    `yaml:"include_common,omitempty"`
	Groups        []yamlGroup `yaml:"groups,omitempty"`
}

type yamlService struct {
	Code  string `yaml:"code"`
	Title string `yaml:"title"`
}

type yamlServicesFile struct {
	Services []yamlService `yaml:"services"`
}

type yamlCommonFile struct {
	Groups map[string]yamlGroup `yaml:"groups"`
}

type yamlRoutingFile struct {
	Rules []struct {
		Match struct {
			Type string `yaml:"type"`
		} `yaml:"match"`
		Always []string `yaml:"always"`
		When   []struct {
			TagsAny []string `yaml:"tags_any"`
			Add     []string `yaml:"add"`
		} `yaml:"when"`
	} `yaml:"rules"`
}

// LoadCatalog reads multi-file YAML catalog from dir into memory.
func LoadCatalog(dir string) (*yamlCatalog, error) {
	commonRaw, err := os.ReadFile(filepath.Join(dir, "common_tags.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read common_tags: %w", err)
	}
	var common yamlCommonFile
	if err := yaml.Unmarshal(commonRaw, &common); err != nil {
		return nil, fmt.Errorf("parse common_tags: %w", err)
	}

	servicesRaw, err := os.ReadFile(filepath.Join(dir, "services.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read services: %w", err)
	}
	var servicesFile yamlServicesFile
	if err := yaml.Unmarshal(servicesRaw, &servicesFile); err != nil {
		return nil, fmt.Errorf("parse services: %w", err)
	}

	typesDir := filepath.Join(dir, "types")
	entries, err := os.ReadDir(typesDir)
	if err != nil {
		return nil, fmt.Errorf("read types dir: %w", err)
	}

	c := &yamlCatalog{
		typeBy:  make(map[string]*loadedType),
		svcBy:   make(map[string]models.EmergencyService),
		routeBy: make(map[string]*routingRule),
	}

	for _, s := range servicesFile.Services {
		if s.Code == "" {
			return nil, fmt.Errorf("service with empty code")
		}
		if _, ok := c.svcBy[s.Code]; ok {
			return nil, fmt.Errorf("duplicate service code %q", s.Code)
		}
		svc := models.EmergencyService{Code: s.Code, Title: s.Title}
		c.services = append(c.services, svc)
		c.svcBy[s.Code] = svc
	}
	sort.Slice(c.services, func(i, j int) bool {
		return c.services[i].Title < c.services[j].Title
	})

	var loaded []loadedType
	seenTypes := make(map[string]struct{})
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(typesDir, e.Name()))
		if err != nil {
			return nil, err
		}
		var tf yamlTypeFile
		if err := yaml.Unmarshal(raw, &tf); err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		if tf.Code == "" {
			return nil, fmt.Errorf("%s: empty type code", e.Name())
		}
		if _, ok := seenTypes[tf.Code]; ok {
			return nil, fmt.Errorf("duplicate type code %q", tf.Code)
		}
		seenTypes[tf.Code] = struct{}{}

		groups := make([]yamlGroup, 0, len(tf.Groups)+len(tf.IncludeCommon))
		groups = append(groups, tf.Groups...)
		for _, key := range tf.IncludeCommon {
			cg, ok := common.Groups[key]
			if !ok {
				return nil, fmt.Errorf("type %s: unknown include_common %q", tf.Code, key)
			}
			groups = append(groups, cg)
		}

		lt := loadedType{IncidentType: models.IncidentType{Code: tf.Code, Title: tf.Title}}
		tagCodes := make(map[string]struct{})
		for i, g := range groups {
			mode, ok := models.ParseTagSelectionMode(g.SelectionMode)
			if !ok {
				if g.SelectionMode == "" {
					mode = models.TagSelectionMulti
				} else {
					return nil, fmt.Errorf("type %s group %s: invalid selection_mode %q", tf.Code, g.Code, g.SelectionMode)
				}
			}
			grp := models.IncidentTagGroup{
				IncidentTypeCode: tf.Code,
				Code:             g.Code,
				Title:            g.Title,
				SelectionMode:    mode,
				ParentTagCode:    g.ParentTag,
				SortOrder:        i,
			}
			for j, t := range g.Tags {
				if t.Code == "" {
					return nil, fmt.Errorf("type %s group %s: empty tag code", tf.Code, g.Code)
				}
				if _, ok := tagCodes[t.Code]; ok {
					return nil, fmt.Errorf("type %s: duplicate tag code %q", tf.Code, t.Code)
				}
				tagCodes[t.Code] = struct{}{}
				grp.Tags = append(grp.Tags, models.IncidentTag{
					IncidentTypeCode: tf.Code,
					GroupCode:        g.Code,
					Code:             t.Code,
					Title:            t.Title,
					SortOrder:        j,
				})
			}
			lt.Groups = append(lt.Groups, grp)
		}
		if err := validateParents(lt); err != nil {
			return nil, err
		}
		loaded = append(loaded, lt)
	}
	sort.Slice(loaded, func(i, j int) bool {
		return loaded[i].Title < loaded[j].Title
	})
	c.loaded = loaded
	for i := range c.loaded {
		c.types = append(c.types, c.loaded[i].IncidentType)
		c.typeBy[c.loaded[i].Code] = &c.loaded[i]
	}

	routingPath := filepath.Join(dir, "routing.yaml")
	if raw, err := os.ReadFile(routingPath); err == nil {
		var rf yamlRoutingFile
		if err := yaml.Unmarshal(raw, &rf); err != nil {
			return nil, fmt.Errorf("parse routing: %w", err)
		}
		for _, r := range rf.Rules {
			if r.Match.Type == "" {
				return nil, fmt.Errorf("routing rule with empty type")
			}
			if _, ok := c.typeBy[r.Match.Type]; !ok {
				return nil, fmt.Errorf("routing: unknown type %q", r.Match.Type)
			}
			for _, code := range r.Always {
				if _, ok := c.svcBy[code]; !ok {
					return nil, fmt.Errorf("routing type %s: unknown service %q", r.Match.Type, code)
				}
			}
			rule := routingRule{TypeCode: r.Match.Type, Always: append([]string(nil), r.Always...)}
			for _, w := range r.When {
				for _, code := range w.Add {
					if _, ok := c.svcBy[code]; !ok {
						return nil, fmt.Errorf("routing type %s: unknown service %q", r.Match.Type, code)
					}
				}
				rule.When = append(rule.When, whenRule{
					TagsAny: append([]string(nil), w.TagsAny...),
					Add:     append([]string(nil), w.Add...),
				})
			}
			c.routing = append(c.routing, rule)
			c.routeBy[rule.TypeCode] = &c.routing[len(c.routing)-1]
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read routing: %w", err)
	}

	return c, nil
}

func validateParents(lt loadedType) error {
	tags := make(map[string]struct{})
	for _, g := range lt.Groups {
		for _, t := range g.Tags {
			tags[t.Code] = struct{}{}
		}
	}
	for _, g := range lt.Groups {
		if g.ParentTagCode == "" {
			continue
		}
		if _, ok := tags[g.ParentTagCode]; !ok {
			return fmt.Errorf("type %s group %s: unknown parent_tag %q", lt.Code, g.Code, g.ParentTagCode)
		}
	}
	return nil
}

func (c *yamlCatalog) recommendServices(typeCode string, tagCodes []string) []string {
	rule, ok := c.routeBy[typeCode]
	if !ok {
		return nil
	}
	selected := make(map[string]struct{}, len(tagCodes))
	for _, t := range tagCodes {
		selected[t] = struct{}{}
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(codes []string) {
		for _, code := range codes {
			if _, ok := seen[code]; ok {
				continue
			}
			seen[code] = struct{}{}
			out = append(out, code)
		}
	}
	add(rule.Always)
	for _, w := range rule.When {
		for _, t := range w.TagsAny {
			if _, ok := selected[t]; ok {
				add(w.Add)
				break
			}
		}
	}
	return out
}
