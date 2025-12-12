#!/bin/bash
/usr/bin/echo "Click into the browser address bar or somewhere"
/usr/bin/sleep 3s
./dist/nvclient -addr 192.168.8.244:60768 -device-id roberts-phone-2 -key-hex a9230fb4fb086fe41fc1d24b32c7aa47000dd7bfb7fdb51e83100610be37feee -password "SuperStrongPassword123!"
