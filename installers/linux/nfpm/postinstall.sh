#!/bin/sh
set -e

echo "NovaKey installed."
echo ""
echo "Per-user setup (run as your normal user):"
echo "  mkdir -p ~/.local/share/novakey ~/.config/novakey"
echo "  cp -n /usr/share/novakey/server_config.yaml ~/.local/share/novakey/server_config.yaml"
echo "  cp -n /usr/share/novakey/server_config.yaml ~/.config/novakey/server_config.yaml"
echo "  systemctl --user daemon-reload"
echo "  systemctl --user enable --now novakey"
echo ""
echo "Optional (run service without login):"
echo "  sudo loginctl enable-linger \$USER"

