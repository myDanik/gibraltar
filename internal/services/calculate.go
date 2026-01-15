package services

import (
	"fmt"
	"gibraltar/internal/models"
	"log"
	"sort"
	"time"
)

const TestAttempt int = 3

type Dependencies struct {
	PreparationService *PreparationService
	VlessTestService   *URLTestService
	Cache              *Cache
}

func (d Dependencies) CalculateAvailableServers() {
	start := time.Now()
	defer func() {
		fmt.Println(time.Since(start))
	}()
	err := d.PreparationService.Pull()
	if err != nil {
		log.Println(err)
		if _, ok := d.Cache.Get(AvailableKey); ok {
			return
		}
	}
	configs, err := d.PreparationService.ParseConfigs("/githubmirror/bypass/bypass-all.txt")
	if err != nil {
		panic(err)
	}
	subnets, err := d.PreparationService.GetSubnetsFromFile("/source/config/cidrwhitelist.txt")
	if err != nil {
		panic(err)
	}
	filter := NewConfigFilter(subnets)
	SortSubnets(subnets)
	out := configs[:0]
	for _, c := range configs {
		ok, _ := filter.IsIPFromWhitelist(c.IP)
		if ok {
			out = append(out, c)
		}
	}
	configs = out
	var beforeTest []models.VlessConfig
	copy(configs, beforeTest)
	d.Cache.Set(AllKey, beforeTest)
	for i := 0; i < TestAttempt; i++ {
		TestConfigs(configs, d.VlessTestService)
	}

	d.Cache.Set(AvailableKey, configs[:SortConfigsByTestResult(configs)])

}

func SortConfigsByTestResult(configs []models.VlessConfig) (availableCount int) {
	sort.Slice(configs, func(i, j int) bool {
		ai := configs[i].TestResult
		aj := configs[j].TestResult

		vi := ai > 0
		vj := aj > 0

		if vi != vj {
			return vi
		}

		if vi && vj {
			return ai < aj
		}

		return false
	})

	availableCount = 0
	for _, v := range configs {
		if v.TestResult > -1 {
			availableCount++
		}
	}
	return availableCount
}
