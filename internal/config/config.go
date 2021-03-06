package config

import (
	"crypto/rand"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type (
	// Config holds the configuration
	Config struct {
		Listen        string `envconfig:"LISTEN" default:":8080"`
		URL           string `envconfig:"URL" default:"http://localhost:8080"`
		PrometheusURL string `envconfig:"PROMETHEUS_URL" default:"http://localhost:9090"`
		CookieSecret  []byte `envconfig:"COOKIE_SECRET"`

		AuthProvider     string `envconfig:"AUTH_PROVIDER" default:"oidc"`
		OidcIssuer       string `envconfig:"OIDC_ISSUER" required:"true"`
		OidcClientID     string `envconfig:"OIDC_CLIENT_ID" required:"true"`
		OidcClientSecret string `envconfig:"OIDC_CLIENT_SECRET" required:"true"`
		OidcRolesClaim   string `envconfig:"OIDC_ROLES_CLAIM" default:"roles"`

		ACLFile string `envconfig:"ACL_FILE" default:"prometheus-oidc.yml"`
		OIDCMap  interface{}
	}
)

// Parse parses the environment for configuration and the provided configuration file for ACLs
func Parse() (c *Config, err error) {
	// handle environment
	c = &Config{}
	err = envconfig.Process("", c)
	if err != nil {
		return nil, err
	}

	switch l := len(c.CookieSecret); l {
	case 64:
		fallthrough
	case 32:
		log.Info("cookie secret provided via env")
	case 0:
		log.Warn("no cookie secret provided, generating a random one")
		c.CookieSecret = make([]byte, 64)
		_, err = rand.Read(c.CookieSecret)
		if err != nil {
			return nil, fmt.Errorf("unable to generate secret key: %s", err)
		}
	default:
		return nil, fmt.Errorf("unable to use provided secret key with %d bytes, use 32 or 64", l)
	}

	// handle config
	fp, err := os.Open(c.ACLFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open config: %s", err)
	}
	OIDCMapLoad := map[string]map[string]interface{}{}
	err = yaml.NewDecoder(fp).Decode(&OIDCMapLoad)
	if err != nil {
		return nil, fmt.Errorf("unable to load config: %s", err)
	}
	c.OIDCMap = OIDCMap{}
	for role, aclLoad := range OIDCMapLoad {
		role := OidcRole(role)
		_, ok := c.OIDCMap[role]
		if !ok {
			c.OIDCMap[role] = &ACL{
				Named: NamedACL{},
				Regex: []RegexACL{},
			}
		}
		loadInto := c.OIDCMap[role]
		for metricName, query := range aclLoad {
			err = loadInto.ParseAndStoreACL(metricName, query)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}