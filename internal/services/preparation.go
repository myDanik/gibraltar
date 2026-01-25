package services

import (
	"bufio"
	"errors"
	"gibraltar/internal/models"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type PreparationService struct {
	LocalPath string
	RemoteURL string
	Branch    string
}

func NewPreparationService(localPath, remoteURL, branch string) *PreparationService {
	return &PreparationService{
		LocalPath: localPath,
		RemoteURL: remoteURL,
		Branch:    branch,
	}
}
func (s PreparationService) Pull() error {
	if _, err := os.Stat(s.LocalPath); os.IsNotExist(err) {
		return s.cloneRepo()
	}

	r, err := git.PlainOpen(s.LocalPath)
	if err != nil {
		_ = os.RemoveAll(s.LocalPath)
		return s.cloneRepo()
	}

	if err := r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/*:refs/remotes/origin/*")},
		Force:      true,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		_ = os.RemoveAll(s.LocalPath)
		return s.cloneRepo()
	}

	remoteRef := plumbing.NewRemoteReferenceName("origin", s.Branch)
	ref, err := r.Reference(remoteRef, true)
	if err != nil {
		_ = os.RemoveAll(s.LocalPath)
		return s.cloneRepo()
	}

	w, err := r.Worktree()
	if err != nil {
		_ = os.RemoveAll(s.LocalPath)
		return s.cloneRepo()
	}

	if err := w.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: ref.Hash(),
	}); err != nil {
		_ = os.RemoveAll(s.LocalPath)
		return s.cloneRepo()
	}

	return nil
}

func (s PreparationService) cloneRepo() error {
	if err := os.MkdirAll(filepath.Dir(s.LocalPath), 0o755); err != nil {
		return err
	}

	_, err := git.PlainClone(s.LocalPath, false, &git.CloneOptions{
		URL:           s.RemoteURL,
		ReferenceName: plumbing.NewBranchReferenceName(s.Branch),
		SingleBranch:  true,
		Depth:         1,
	})
	return err
}

func (s PreparationService) ParseConfigs(inDirectoryPath string) ([]models.VlessConfig, error) {
	file, err := os.Open(s.LocalPath + inDirectoryPath)
	if err != nil {
		return nil, err

	}
	defer file.Close()
	configs := make([]models.VlessConfig, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.ReplaceAll(line, "&amp;", "&")
		if line == "" {
			continue
		}
		config := models.VlessConfig{
			URL: line,
		}
		err = halfParseVlessURL(&config)
		if err != nil {
			continue
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func halfParseVlessURL(config *models.VlessConfig) error {
	u, err := url.Parse(config.URL)
	if err != nil {
		return err
	}
	if u.Scheme != "vless" {
		return errors.New("not vless url")
	}

	q := u.Query()
	address := u.Hostname()
	if !validateIP(address) {
		return errors.New("not valid ip address")
	}
	config.IP = address
	config.Port = u.Port()
	config.SNI = q.Get("sni")
	return nil
}

func validateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil
}

func (s PreparationService) GetSubnetsFromFile(inDirectoryPath string) ([][]byte, error) {
	file, err := os.Open(s.LocalPath + inDirectoryPath)
	if err != nil {
		return nil, err
	}
	subnetsMap := make(map[string]int)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		subnetsMap[string(getSubnet(line))]++
	}
	var resultList [][]byte
	for i := range subnetsMap {
		resultList = append(resultList, []byte(i))
	}
	return resultList, nil
}

func setTestResultValue(config *models.VlessConfig, localPort int, service *URLTestService) {
	if config.TestResult > 0 {
		return
	}
	time, err := TLSTest(config.IP, config.Port, config.SNI, 2*time.Second)
	if time <= 0 || err != nil {
		config.TestResult = -1
		return
	}

	lat, err := service.Test(config.URL, localPort)
	if err != nil {
		config.TestResult = -1
	} else {
		config.TestResult = int(lat.Milliseconds())
	}

}

func TestConfigs(configs []models.VlessConfig, service *URLTestService) {
	log.Printf("%d configs will be tested\n", len(configs))
	ports := make([]int, 0)
	for i := 2081; i <= 2091; i++ {
		ports = append(ports, i)
	}
	jobs := make(chan int)

	var wg sync.WaitGroup

	for w := 0; w < len(ports); w++ {
		wg.Add(1)
		port := ports[w]

		go func(p int) {
			defer wg.Done()
			for idx := range jobs {
				setTestResultValue(&configs[idx], p, service)
			}
		}(port)
	}

	for i := range configs {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
}
