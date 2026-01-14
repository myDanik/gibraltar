package services

import (
	"fmt"
	"gibraltar/internal/models"
	"log"
	"strconv"
	"strings"
)

type ConfigFilter struct {
	AllowedSubnets [][]byte
}

func NewConfigFilter(allowedSubnets [][]byte) *ConfigFilter {
	return &ConfigFilter{
		AllowedSubnets: allowedSubnets,
	}
}

func (f ConfigFilter) IsAvailableConfig(config *models.VlessConfig) (bool, error) {
	ok, err := f.IsIPFromWhitelist(config.IP)
	if !ok {
		return false, err
	}
	return true, nil
}

func (f ConfigFilter) IsIPFromWhitelist(ip string) (bool, error) {
	ipSubnet := getSubnet(ip)
	left := 0
	right := len(f.AllowedSubnets) - 1
	for left <= right {
		mid := left + (right-left)/2
		compResult := compareIPs(ipSubnet, f.AllowedSubnets[mid])
		if compResult < 0 {
			right = mid - 1
		} else if compResult > 0 {
			left = mid + 1
		} else {
			return true, nil
		}
	}
	return false, fmt.Errorf("ip not exist in whitelist")
}

func compareIPs(ip1, ip2 []byte) int {
	ip1SplittedString := strings.Split(string(ip1), ".")
	ip2SplittedString := strings.Split(string(ip2), ".")
	var ip1Splitted [3]int
	var ip2Splitted [3]int
	var err error
	for i := 0; i < 3; i++ {
		ip1Splitted[i], err = strconv.Atoi(ip1SplittedString[i])
		if err != nil {
			log.Println(err)
		}
		ip2Splitted[i], err = strconv.Atoi(ip2SplittedString[i])
	}

	if ip1Splitted[0] < ip2Splitted[0] {
		return -1
	} else if ip1Splitted[0] > ip2Splitted[0] {
		return 1
	}

	if ip1Splitted[1] < ip2Splitted[1] {
		return -1
	} else if ip1Splitted[1] > ip2Splitted[1] {
		return 1
	}

	if ip1Splitted[2] < ip2Splitted[2] {
		return -1
	} else if ip1Splitted[2] > ip2Splitted[2] {
		return 1
	}

	return 0
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

func SortSubnets(subnets [][]byte) {
	quickSort(subnets, 0, len(subnets)-1)
}

func quickSort(subnets [][]byte, left, right int) {
	if left >= right {
		return
	}

	pivotIndex := left + (right-left)/2
	pivotValue := subnets[pivotIndex]

	i, j := left, right
	for i <= j {
		for compareIPs(subnets[i], pivotValue) < 0 {
			i++
		}
		for compareIPs(subnets[j], pivotValue) > 0 {
			j--
		}

		if i <= j {
			subnets[i], subnets[j] = subnets[j], subnets[i]
			i++
			j--
		}
	}

	if left < j {
		quickSort(subnets, left, j)
	}
	if i < right {
		quickSort(subnets, i, right)
	}
}
