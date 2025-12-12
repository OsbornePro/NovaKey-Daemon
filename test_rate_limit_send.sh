#!/bin/bash
/usr/bin/echo "Click into the browser address bar or somewhere"
/usr/bin/sleep 3s
for i in {1..71}; do
  ./dist/nvclient \
    -addr 127.0.0.1:60768 \
    -device-id roberts-phone \
    -key-hex 7f0c9e6b3a8d9c0b9a45f32caf51bc0f7a83f663e27aa4b4ca9e5216a28e1234 \
    -password "SuperStrongPassword123!@#"
done

