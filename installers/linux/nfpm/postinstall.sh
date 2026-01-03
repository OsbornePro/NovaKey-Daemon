#!/bin/sh
set -e

echo "NovaKey installed."
echo ""
echo "Enable NovaKey-Daemon for your user:"
echo "  systemctl --user enable --now novakey"
echo ""
echo "Optional (run without login):"
echo "  sudo loginctl enable-linger \$USER"

