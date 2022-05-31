package configs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

type UserAuth struct {
	CertPath      string
	KeyPath       string
	Username      string
	Password      string
}

type Configs struct {
	ZonefilesPath     string
	EtcdKeyPrefix     string
	EtcdEndpoints     string
	CaCertPath        string
	UserAuth          UserAuth
	ConnectionTimeout uint64
	RequestTimeout    uint64
	RequestRetries    uint64
}

func getEnv(key string, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}

func checkConfigsIntegrity(c Configs) error {
	if c.ZonefilesPath == "" {
		return errors.New("Configuration error: Zone file path cannot be empty")
	}

	if c.EtcdEndpoints == "" {
		return errors.New("Configuration error: Etcd endpoints cannot be empty")
	}

	if c.CaCertPath == "" {
		return errors.New("Configuration error: CA certificate path cannot be empty")
	}

	noValidAuth := (c.UserAuth.CertPath == "" || c.UserAuth.KeyPath == "") && (c.UserAuth.Username == "" || c.UserAuth.Password == "")
	ambiguousAuthMethod := (c.UserAuth.CertPath != "" || c.UserAuth.KeyPath != "") && (c.UserAuth.Username != "" || c.UserAuth.Password != "")

	if noValidAuth || ambiguousAuthMethod {
		return errors.New("Configuration error: Either user certificate AND key path should not be empty XOR user name AND password should not be empty")
	}

	if c.EtcdKeyPrefix == "" {
		return errors.New("Configuration error: Etcd key prefix cannot be empty")
	}

	return nil
}

func GetConfigs() (Configs, error) {
	var c Configs
	_, err := os.Stat("./configs.json")

	if err == nil {
		bs, err := ioutil.ReadFile("./configs.json")
		if err != nil {
			return Configs{}, errors.New(fmt.Sprintf("Error reading configuration file: %s", err.Error()))
		}
	
		err = json.Unmarshal(bs, &c)
		if err != nil {
			return Configs{}, errors.New(fmt.Sprintf("Error reading configuration file: %s", err.Error()))
		}
	} else if errors.Is(err, os.ErrNotExist) {
		var connectionTimeout, requestTimeout, requestRetries uint64
		var err error
		connectionTimeout, err = strconv.ParseUint(getEnv("CONNECTION_TIMEOUT", "0"), 10, 64)
		if err != nil {
			return Configs{}, errors.New("Error fetching configuration environment variables: CONNECTION_TIMEOUT must be an unsigned integer")
		}
		requestTimeout, err = strconv.ParseUint(getEnv("REQUEST_TIMEOUT", "0"), 10, 64)
		if err != nil {
			return Configs{}, errors.New("Error fetching configuration environment variables: REQUEST_TIMEOUT must be an unsigned integer")
		}
		requestRetries, err = strconv.ParseUint(getEnv("REQUEST_RETRIES", "0"), 10, 64)
		if err != nil {
			return Configs{}, errors.New("Error fetching configuration environment variables: REQUEST_RETRIES must be an unsigned integer")
		}
		c.ConnectionTimeout = connectionTimeout
		c.RequestTimeout = requestTimeout
		c.RequestRetries = requestRetries

		userAuth := UserAuth{
			CertPath: getEnv("USER_CERT_PATH", ""),
			KeyPath: getEnv("USER_KEY_PATH", ""),
			Username: getEnv("USER_NAME", ""),
			Password: getEnv("USER_PASSWORD", ""),
		}
		c.UserAuth = userAuth

		c.ZonefilesPath = os.Getenv("ZONEFILE_PATH")
		c.EtcdEndpoints = os.Getenv("ETCD_ENDPOINTS")
		c.CaCertPath = os.Getenv("CA_CERT_PATH")
		c.EtcdKeyPrefix = os.Getenv("ETCD_KEY_PREFIX")
	} else {
		return Configs{}, errors.New(fmt.Sprintf("Error reading configuration file: %s", err.Error()))
	}

	err = checkConfigsIntegrity(c)
	if err != nil {
		return Configs{}, err
	}

	return c, nil
}