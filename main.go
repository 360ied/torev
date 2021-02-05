package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
)

type configJSON struct {
	KeyBase64   string // Private Key to use for the TOR Hidden Service. Must be a v3 (ed25519) key. Must be encoded in Base64
	RemotePorts []int  // Ports exposed in the TOR Hidden Service
	LocalPort   int    // The port the connections should be proxied to
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config-path", "config.json", "Path to the configuration file")
	flag.Parse()

	var (
		key         ed25519.PrivateKey
		remotePorts []int
		localPort   int
	)

	configFile, configFileErr := os.Open(configPath)
	if configFileErr != nil {
		if errors.Is(configFileErr, os.ErrNotExist) {
			// file does not exist, so we create it and generate a default config
			log.Fatalf("[WARN] Config file doesn't exist, so creating one at %s", configPath)

			var generateKeyErr error
			_, key, generateKeyErr = ed25519.GenerateKey(nil)
			if generateKeyErr != nil {
				log.Fatalf("[FATAL] Failed to generate private key: %v", generateKeyErr)
			}

			remotePorts = []int{80}
			localPort = 8080

			marshalledConfig, marshalledConfigErr := json.Marshal(configJSON{
				KeyBase64:   base64.StdEncoding.EncodeToString(key),
				RemotePorts: remotePorts,
				LocalPort:   localPort,
			})
			if marshalledConfigErr != nil {
				log.Fatalf("[FATAL] Failed to marshal config: %v", marshalledConfigErr)
			}

			var configFileCreateErr error
			configFile, configFileCreateErr = os.Create(configPath)
			if configFileCreateErr != nil {
				log.Fatalf("[FATAL] Failed to create the config file (path: %s): %v", configPath, configFileCreateErr)
			}

			if _, err := configFile.Write(marshalledConfig); err != nil {
				log.Fatalf("[FATAL] Failed to write generated config to config file (path: %s): %v", configPath, configFileCreateErr)
			}
		} else {
			log.Fatalf("[FATAL] Failed to open or create the config file (path: %s): %v", configPath, configFileErr)
		}
	} else { // no error
		configFileData, configFileDataErr := ioutil.ReadAll(configFile)
		if configFileDataErr != nil {
			log.Fatalf("[FATAL] Failed to read config file (path: %s): %v", configPath, configFileDataErr)
		}

		var unmarshalledConfig configJSON
		if err := json.Unmarshal(configFileData, &unmarshalledConfig); err != nil {
			log.Fatalf("[FATAL] Failed to unmarshal config file (path: %s): %v", configPath, configFileDataErr)
		}

		var keyDecodeErr error
		key, keyDecodeErr = base64.StdEncoding.DecodeString(unmarshalledConfig.KeyBase64)
		if keyDecodeErr != nil {
			log.Fatalf("[FATAL] Failed to decode private key from base64: %v", keyDecodeErr)
		}

		remotePorts = unmarshalledConfig.RemotePorts
		localPort = unmarshalledConfig.LocalPort
	}
}
