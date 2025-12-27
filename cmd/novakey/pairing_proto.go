func handlePairConn(conn net.Conn) error {
    defer conn.Close() // IMPORTANT: ensure iOS sees EOF after ACK

    if serverDecapKey == nil || len(serverEncapKey) == 0 {
        return fmt.Errorf("server keys not initialized")
    }

    // Hard timeout for pairing flow; clear before return.
    _ = conn.SetDeadline(time.Now().Add(25 * time.Second))
    defer func() { _ = conn.SetDeadline(time.Time{}) }()

    ip := remoteIP(conn)
    if !allowPairHelloFromIP(ip) {
        return fmt.Errorf("pair hello rate limited for %s", ip)
    }

    br := bufio.NewReaderSize(conn, 8192)

    // Read hello JSON line
    line, err := br.ReadBytes('\n')
    if err != nil {
        return fmt.Errorf("read hello: %w", err)
    }

    var hello pairHello
    if err := json.Unmarshal(trimNL(line), &hello); err != nil {
        return fmt.Errorf("bad hello json: %w", err)
    }
    if hello.Op != "hello" || hello.V != 1 {
        return fmt.Errorf("unexpected hello op/v")
    }
    if hello.Token == "" {
        return fmt.Errorf("missing token")
    }

    // Validate + consume token (one-time)
    tokenBytes, err := consumePairToken(hello.Token)
    if err != nil {
        return err
    }

    // Send server key info (plaintext)
    fp16 := fp16Hex(serverEncapKey)
    resp := pairServerKey{
        Op:          "server_key",
        V:           1,
        KID:         "1",
        KyberPubB64: base64.StdEncoding.EncodeToString(serverEncapKey),
        FP16Hex:     fp16,
        ExpiresUnix: time.Now().Add(2 * time.Minute).Unix(),
    }
    b, _ := json.Marshal(resp)
    b = append(b, '\n')

    _ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
    if _, err := conn.Write(b); err != nil {
        return fmt.Errorf("write server_key: %w", err)
    }
    _ = conn.SetWriteDeadline(time.Time{})

    // Now read binary encapsulated register frame.
    ct, nonce, ciphertext, err := readPairBinaryFrame(br)
    if err != nil {
        return err
    }
    if len(ct) != mlkem768.CiphertextSize {
        return fmt.Errorf("bad ct len: %d", len(ct))
    }

    sharedKem, err := mlkem768.Decapsulate(serverDecapKey, ct)
    if err != nil {
        return fmt.Errorf("decaps failed: %w", err)
    }

    aeadKey, err := derivePairAEADKey(sharedKem, tokenBytes)
    if err != nil {
        return err
    }
    aead, err := chacha20poly1305.NewX(aeadKey)
    if err != nil {
        return fmt.Errorf("NewX: %w", err)
    }

    aad := makePairAAD(ct, nonce)

    plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
    if err != nil {
        return fmt.Errorf("pair decrypt failed: %w", err)
    }

    var reg pairRegister
    if err := json.Unmarshal(plaintext, &reg); err != nil {
        return fmt.Errorf("bad register json: %w", err)
    }
    if reg.Op != "register" || reg.V != 1 {
        return fmt.Errorf("unexpected register op/v")
    }

    if reg.DeviceID == "" {
        reg.DeviceID = "ios-" + randHex(8)
    }

    if cfg.RotateDevicePSKOnRepair {
        devicesMu.RLock()
        _, exists := devices[reg.DeviceID]
        devicesMu.RUnlock()
        if exists {
            reg.DeviceKeyHex = ""
        }
    }

    if reg.DeviceKeyHex == "" {
        reg.DeviceKeyHex = randHex(32)
    }

    if err := writeDevicesFile(cfg.DevicesFile, reg.DeviceID, reg.DeviceKeyHex); err != nil {
        return fmt.Errorf("write devices: %w", err)
    }
    if err := reloadDevicesFromDisk(); err != nil {
        return fmt.Errorf("reload devices: %w", err)
    }

    ack := map[string]any{
        "op":        "ok",
        "v":         1,
        "device_id": reg.DeviceID,
    }
    ackB, _ := json.Marshal(ack)

    ackNonce := make([]byte, aead.NonceSize())
    _, _ = rand.Read(ackNonce)
    ackCT := aead.Seal(nil, ackNonce, ackB, makePairAAD(ct, ackNonce))

    _ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
    if err := writePairAck(conn, ackNonce, ackCT); err != nil {
        return fmt.Errorf("write ack: %w", err)
    }
    _ = conn.SetWriteDeadline(time.Time{})

    // Help the client finish reading immediately
    if tcp, ok := conn.(*net.TCPConn); ok {
        _ = tcp.CloseWrite()
    }

    log.Printf("[pair] paired device_id=%s (saved + reloaded)", reg.DeviceID)
    log.Printf("[pair] wrote ack bytes=%d", len(ackNonce)+len(ackCT))
    return nil
}
