package config

import (
	"os"
	"path/filepath"
	"time"
)

// Stability calculation parameters
const (
	Gain    = 1.5
	Decay   = 0.7
	P       = 1.5
	Q       = 1.3
	MinDrop = 2
	MAX     = 100
)

// Stability criteria
const (
	MinValueForAccept float64 = 5.0
	MinValueForStable float64 = 50.0
)

// Cache Keys
const (
	AvailableKey = "latestResults"
	AllKey       = "currentConfigs"
)

// Directory and files
var (
	ConfigSourceDirectoryPath = filepath.Join(os.Getenv("HOME"), "rjsxrd")
	VlessSecureConfigs        = filepath.Join(ConfigSourceDirectoryPath, "githubmirror", "bypass", "bypass-all.txt")
	CIDRWhitelist             = filepath.Join(ConfigSourceDirectoryPath, "source", "config", "cidrwhitelist.txt")
	URLsWhitelist             = filepath.Join(ConfigSourceDirectoryPath, "source", "config", "whitelist-all.txt")
)

// Remote git repo
const (
	RemoteRepository = "https://github.com/whoahaow/rjsxrd.git"
	RemoteBranch     = "main"
)

// Sing-box workers
const (
	WorkersCount = 40
)

// Test server (URL)
const (
	TestURL = "http://cp.cloudflare.com/"
)

// Timings
const (
	PullCooldown = time.Minute * 5
)
