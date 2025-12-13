// cmd/nvpair/main.go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/chacha20poly1305"
    "github.com/skip2/go-qrcode"
)

type deviceConfig struct {
	ID     string `json:"id"`
	KeyHex string `json:"key_hex"`
}

type devicesConfigFile struct {
	Devices []deviceConfig `json:"devices"`
}

type serverConfig struct {
    ListenAddr     string `json:"listen_addr"`
    DevicesFile    string `json:"devices_file"`
    ServerKeysFile string `json:"server_keys_file"`
}

type serverKeys struct {
    KyberPub  string `json:"kyber768_public"`
    KyberPriv string `json:"kyber768_secret"`
}

type pairingInfo struct {
    Version           int    `json:"v"`
    DeviceID          string `json:"device_id"`
    DeviceKeyHex      string `json:"device_key_hex"`
    ServerAddr        string `json:"server_addr"`
    ServerKyber768Pub string `json:"server_kyber768_pub"`
}

var (
    devicesFileFlag   = flag.String("devices-file", "devices.json", "path to devices.json")
    configFileFlag    = flag.String("config-file", "server_config.json", "path to server_config.json")
    deviceIDFlag      = flag.String("id", "", "device ID to add or update (required)")
    forceFlag         = flag.Bool("force", false, "overwrite existing device with same ID")
    qrFlag            = flag.Bool("qr", true, "render pairing info as an ASCII QR code")
)

func main() {
    flag.Parse()

    if *deviceIDFlag == "" {
        fmt.Fprintln(os.Stderr, "ERROR: -id is required (device ID)")
        flag.Usage()
        os.Exit(1)
    }

    // 1. Load or create devices.json
    devicesPath := *devicesFileFlag
    absDevices, _ := filepath.Abs(devicesPath)

    cfg, err := loadDevices(devicesPath)
    if err != nil {
        if os.IsNotExist(err) {
            fmt.Printf("devices file %s does not exist, creating new one\n", absDevices)
            cfg = &devicesConfigFile{Devices: []deviceConfig{}}
        } else {
            fmt.Fprintf(os.Stderr, "ERROR: loading devices file %s: %v\n", absDevices, err)
            os.Exit(1)
        }
    }

    // 2. Generate new random per-device key
    keyBytes := make([]byte, chacha20poly1305.KeySize)
    if _, err := rand.Read(keyBytes); err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: rand.Read key: %v\n", err)
        os.Exit(1)
    }
    keyHex := hex.EncodeToString(keyBytes)

    // 3. Add or update device ID
    existingIdx := -1
    for i, d := range cfg.Devices {
        if d.ID == *deviceIDFlag {
            existingIdx = i
            break
        }
    }

    if existingIdx >= 0 && !*forceFlag {
        fmt.Fprintf(os.Stderr, "ERROR: device ID %q already exists in %s (use -force to overwrite)\n",
            *deviceIDFlag, absDevices)
        os.Exit(1)
    }

    if existingIdx >= 0 && *forceFlag {
        cfg.Devices[existingIdx].KeyHex = keyHex
        fmt.Printf("Updated existing device %q in %s\n", *deviceIDFlag, absDevices)
    } else if existingIdx == -1 {
        cfg.Devices = append(cfg.Devices, deviceConfig{
            ID:     *deviceIDFlag,
            KeyHex: keyHex,
        })
        fmt.Printf("Added new device %q to %s\n", *deviceIDFlag, absDevices)
    }

    if err := saveDevices(devicesPath, cfg); err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: saving devices file %s: %v\n", absDevices, err)
        os.Exit(1)
    }

    // 4. Load server_config.json to get listen_addr and server_keys_file
    serverCfg, err := loadServerConfig(*configFileFlag)
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: loading server config %s: %v\n", *configFileFlag, err)
        os.Exit(1)
    }

    // 5. Load server_keys.json to get Kyber public key
    keysPath := serverCfg.ServerKeysFile
    if keysPath == "" {
        keysPath = "server_keys.json"
    }
    srvKeys, err := loadServerKeys(keysPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: loading server keys %s: %v\n", keysPath, err)
        fmt.Fprintf(os.Stderr, "Hint: start the novakey daemon once to generate server_keys.json\n")
        os.Exit(1)
    }

    // 6. Build pairing info payload
    pairing := pairingInfo{
        Version:           1,
        DeviceID:          *deviceIDFlag,
        DeviceKeyHex:      keyHex,
        ServerAddr:        serverCfg.ListenAddr,
        ServerKyber768Pub: srvKeys.KyberPub,
    }

    pairingJSON, err := json.MarshalIndent(&pairing, "", "  ")
    if err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: marshal pairing info: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("------------------------------------------------------------")
    fmt.Println(" Pairing info (JSON)")
    fmt.Println("------------------------------------------------------------")
    fmt.Println(string(pairingJSON))
    fmt.Println()

    if *qrFlag {
        // Option A: built-in ASCII QR using go-qrcode
        qr, err := qrcode.New(string(pairingJSON), qrcode.Medium)
        if err != nil {
            fmt.Fprintf(os.Stderr, "ERROR: generating QR: %v\n", err)
            os.Exit(1)
        }
        fmt.Println("ASCII QR (scan with your phone):")
        fmt.Println(qr.ToString(false))

        // Option B: tell the user how to pipe pairing JSON into qrencode
        // fmt.Println("To generate a QR code, you can run:")
        // fmt.Println("  echo '<<above JSON>>' | qrencode -t ansiutf8")
    }

    fmt.Println()
    fmt.Println("Use this pairing info in your phone app to configure NovaKey v3.")
}

func loadDevices(path string) (*devicesConfigFile, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg devicesConfigFile
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

func saveDevices(path string, cfg *devicesConfigFile) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0o600); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}

func loadServerConfig(path string) (*serverConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg serverConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    if cfg.ListenAddr == "" {
        cfg.ListenAddr = "127.0.0.1:60768"
    }
    if cfg.ServerKeysFile == "" {
        cfg.ServerKeysFile = "server_keys.json"
    }
    if cfg.DevicesFile == "" {
        cfg.DevicesFile = "devices.json"
    }
    return &cfg, nil
}

func loadServerKeys(path string) (*serverKeys, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var sk serverKeys
    if err := json.Unmarshal(data, &sk); err != nil {
        return nil, err
    }
    if sk.KyberPub == "" {
        return nil, fmt.Errorf("missing kyber768_public")
    }
    return &sk, nil
}

