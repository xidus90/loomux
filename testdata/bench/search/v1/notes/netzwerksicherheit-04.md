---
titel: X.509
quelle: https://en.wikipedia.org/wiki/X.509
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerksicherheit
---

In cryptography, X.509 is an International Telecommunication Union (ITU) standard defining the format of public key certificates. X.509 certificates are used in many Internet protocols, including TLS/SSL, which is the basis for HTTPS, the secure protocol for browsing the web. They are also used in offline applications, like electronic signatures.
An X.509 certificate binds an identity to a public key using a digital signature. A certificate contains an identity (a hostname, or an organization, or an individual) and a public key (RSA, DSA, ECDSA, ed25519, etc.), and is either signed by a certificate authority (CA) or is self-signed. When a certificate is signed by a trusted certificate authority, or validated by other means, someone holding that certificate can use the public key it contains to establish secure communications with another party, or validate documents digitally signed by the corresponding private key.
X.509 also defines certificate revocation lists, which are a means to distribute information about certificates that have been deemed invalid by a signing authority, as well as a certification path validation algorithm, which allows for certificates to be signed by intermediate CA certificates, which are, in turn, signed by other certificates, eventually reaching a trust anchor.
X.509 is defined by the ITU's "Standardization Sector" (ITU-T's SG17), in ITU-T Study Group 17 and is based on Abstract Syntax Notation One (ASN.1), another ITU-T standard.
