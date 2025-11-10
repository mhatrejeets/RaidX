#!/bin/bash
# Generate self-signed SSL certificates for local testing
# For production, use Let's Encrypt (certbot)

mkdir -p ssl

# Generate private key and certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ssl/key.pem \
  -out ssl/cert.pem \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

echo "SSL certificates generated in ./ssl/"
echo "For Azure production deployment, obtain certificates from:"
echo "  1. Azure Key Vault"
echo "  2. Let's Encrypt (certbot)"
echo "  3. Your organization's certificate provider"
