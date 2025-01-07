package certs

import _ "embed"

//go:embed server.crt
var CertPEM []byte
