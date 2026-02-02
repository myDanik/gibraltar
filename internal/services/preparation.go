package services

import (
	"bufio"
	"gibraltar/internal/models"
	"os"
	"strings"
)

func ParseConfigs(fullPath string) ([]*models.VlessConfig, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err

	}
	defer file.Close()
	configs := make([]*models.VlessConfig, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.ReplaceAll(line, "&amp;", "&")
		if line == "" {
			continue
		}
		config := &models.VlessConfig{
			URL: line,
		}

		configs = append(configs, config)
	}
	return configs, nil
}

func GetSubnetsFromFile(fullPath string) (map[string]struct{}, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	subnetsMap := make(map[string]struct{})
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		subnetsMap[string(getSubnet(line))] = struct{}{}
	}

	return subnetsMap, nil
}

func getSubnet(ip string) []byte {
	num := make([]byte, 0)
	dotCount := 0
	for _, ch := range ip {
		if ch == rune('.') {
			if dotCount == 2 {
				break

			}
			dotCount++

		}
		num = append(num, byte(ch))
	}
	return num
}

func GetSNIsFromFile(fullPath string) (map[string]struct{}, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	allowedSNIs := make(map[string]struct{})
	for scanner.Scan() {
		line := scanner.Text()
		allowedSNIs[line] = struct{}{}
	}
	return allowedSNIs, nil

}
