package services

import (
	"fmt"
	"log"
	"sort"
	"time"
)

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
		if _, ok := d.Cache.Get(СacheKey); ok {
			return
		}
	}
	configs, err := d.PreparationService.ParseConfigs("/githubmirror/bypass/bypass-1.txt")
	if err != nil {
		panic(err)
	}
	subnets, err := d.PreparationService.GetSubnetsFromFile("/source/config/cidrwhitelist.txt")
	if err != nil {
		panic(err)
	}
	filter := NewConfigFilter(subnets)
	SortSubnets(subnets)
	for i := 0; i < len(configs); i++ {
		ok, _ := filter.IsIPFromWhitelist(configs[i].IP)
		if !ok {
			configs = append(configs[:i], configs[i+1:]...)
		}
	}
	log.Printf("%d configs will be tested\n", len(configs))
	TestConfigs(configs, *d.VlessTestService)
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

	availableCount := 0
	for _, v := range configs {
		if v.TestResult > -1 {
			availableCount++
		}
	}
	d.Cache.Set(СacheKey, configs[:availableCount])

}
