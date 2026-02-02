package services

import (
	"gibraltar/config"
	"gibraltar/internal/models"
	"log"
	"strings"
	"time"
)

type ConfigUpdater struct {
	Cache          *Cache
	Filter         *ConfigFilter
	URLTestService *URLTestService
}

func NewConfigUpdater(cache *Cache, filter *ConfigFilter, urlTestService *URLTestService) *ConfigUpdater {
	return &ConfigUpdater{
		Cache:          cache,
		Filter:         filter,
		URLTestService: urlTestService,
	}
}

func (u *ConfigUpdater) RunTest(configs []*models.VlessConfig) {
	start := time.Now()
	defer func() {
		log.Printf("woriking time (test): %s\n", time.Since(start))
	}()
	u.URLTestService.TestConfigs(configs, len(configs)/32)

}

func (u *ConfigUpdater) AddConfigsToCacheFromSource() error {
	configs, err := ParseConfigs(config.VlessSecureConfigs)
	if err != nil {
		return err
	}
	filtered := make([]models.VlessConfig, 0, len(configs)/10)
	for _, config := range configs {
		err = parseVlessURL(config)
		if err != nil {
			continue
		}
		if ok, _ := u.Filter.IsAvailableConfig(config); !ok {
			continue
		}
		filtered = append(filtered, *config)
	}
	prevConfigs, ok := u.Cache.Get(config.AllKey)
	if !ok {
		u.Cache.Set(config.AllKey, filtered)
	} else {
		prevMap := make(map[string]models.VlessConfig, len(prevConfigs))
		for i := range prevConfigs {
			prevMap[prevConfigs[i].URL] = prevConfigs[i]
		}
		for i := range filtered {
			if _, ok := prevMap[filtered[i].URL]; !ok {
				prevConfigs = append(prevConfigs, filtered[i])
			}
		}
		u.Cache.Set(config.AllKey, prevConfigs)

	}

	return nil

}

func (u *ConfigUpdater) AddAvailableConfigsToCache() error {
	configs, ok := u.Cache.Get(config.AllKey)
	if !ok {
		if err := u.AddConfigsToCacheFromSource(); err != nil {
			return err
		}
		configs, _ = u.Cache.Get(config.AllKey)
	}
	pointers := make([]*models.VlessConfig, 0, len(configs))
	for idx := range configs {
		pointers = append(pointers, &configs[idx])
	}
	u.RunTest(pointers)
	availableList := filterConfigsByStability(pointers)

	u.Cache.Set(config.AvailableKey, *availableList)
	return nil
}

func filterConfigsByStability(configs []*models.VlessConfig) *[]models.VlessConfig {
	if configs == nil {
		return nil
	}
	result := make([]models.VlessConfig, 0, 10)
	for idx := range configs {
		if configs[idx].Stability >= config.MinValueForAccept {
			result = append(result, *configs[idx])

		}
		if configs[idx].Stability >= config.MinValueForStable {
			markAsStable(&result[len(result)-1])
		}
	}
	return &result
}

func markAsStable(cfg *models.VlessConfig) {
	if cfg == nil {
		return
	}
	s := cfg.URL
	idx := strings.IndexByte(s, '#')
	if idx == -1 {
		cfg.URL = s + "#Стабильный "
		return
	}
	before := s[:idx+1]
	after := s[idx+1:]

	if strings.HasPrefix(after, "Стабильный | ") || strings.HasPrefix(after, "Stable | ") {
		return
	}

	cfg.URL = before + "Стабильный | " + after
}
