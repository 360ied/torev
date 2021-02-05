package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
)

type configJSON struct {
	KeyBase64    string // Private Key to use for the TOR Hidden Service. Must be a v3 (ed25519) key. Must be encoded in Base64
	RemotePorts  []int  // Ports exposed in the TOR Hidden Service
	LocalAddress string // The address the connections should be proxied to
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config-path", "config.json", "Path to the configuration file")
	flag.Parse()

	var (
		key          ed25519.PrivateKey
		remotePorts  []int
		localAddress string
	)

	configFile, configFileErr := os.Open(configPath)
	if configFileErr != nil {
		if errors.Is(configFileErr, os.ErrNotExist) {
			// file does not exist, so we create it and generate a default config
			log.Printf("[WARN] Config file doesn't exist, so creating one at %s", configPath)

			var generateKeyErr error
			_, key, generateKeyErr = ed25519.GenerateKey(nil)
			if generateKeyErr != nil {
				log.Fatalf("[FATAL] Failed to generate private key: %v", generateKeyErr)
			}

			remotePorts = []int{80}
			localAddress = "127.0.0.1:8080"

			marshalledConfig, marshalledConfigErr := json.MarshalIndent(configJSON{
				KeyBase64:    base64.StdEncoding.EncodeToString(key),
				RemotePorts:  remotePorts,
				LocalAddress: localAddress,
			}, "", "\t")
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
			configFile.Close()
		} else {
			log.Fatalf("[FATAL] Failed to open the config file (path: %s): %v", configPath, configFileErr)
		}
	} else { // config file exists
		configFileData, configFileDataErr := ioutil.ReadAll(configFile)
		if configFileDataErr != nil {
			log.Fatalf("[FATAL] Failed to read config file (path: %s): %v", configPath, configFileDataErr)
		}
		configFile.Close()

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
		localAddress = unmarshalledConfig.LocalAddress
	}

	log.Print("[INFO] Starting TOR...")
	t, tErr := tor.Start(context.Background(), &tor.StartConf{ProcessCreator: libtor.Creator})
	if tErr != nil {
		log.Fatalf("[FATAL] Failed to start TOR process: %v", tErr)
	}
	defer t.Close()
	log.Print("[INFO] Done starting TOR.")

	log.Print("[INFO] Starting the listener...")
	listener, listenerErr := t.Listen(context.Background(), &tor.ListenConf{Key: key, RemotePorts: remotePorts})
	if listenerErr != nil {
		log.Panicf("[PANIC] Failed to start the listener: %v", listenerErr)
	}
	defer listener.Close()
	log.Print("[INFO] Done starting the listener.")

	log.Printf("[STARTED] Proxying connections from %s.onion to %s", listener.ID, localAddress)
	for {
		remoteConn, remoteConnErr := listener.Accept()
		if remoteConnErr != nil {
			log.Panicf("[PANIC] Failed to accept connection: %v", remoteConnErr)
		}
		log.Print("[INFO] New connection from remote established.")

		localConn, localConnErr := net.Dial("tcp", localAddress)
		if localConnErr != nil {
			log.Panicf("[PANIC] Failed to create connection with the local address (%s): %v", localAddress, localConnErr)
		}
		log.Print("[INFO] New connection to local established.")

		go func() {
			log.Print("[INFO] Proxying data from remote to local...")
			_, err := io.Copy(localConn, remoteConn)
			log.Printf("[INFO] Stopped proxying data from one of the connections from remote to local: %v", err)
		}()
		go func() {
			log.Print("[INFO] Proxying data from local to remote...")
			_, err := io.Copy(remoteConn, localConn)
			log.Printf("[INFO] Stopped proxying data from one of the connections from local to remote: %v", err)
		}()
	}
}
