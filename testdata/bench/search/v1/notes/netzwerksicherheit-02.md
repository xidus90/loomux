---
titel: Public key certificate
quelle: https://en.wikipedia.org/wiki/Public_key_certificate
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerksicherheit
---

In cryptography, a public-key certificate, also known as a digital certificate or identity certificate, is an electronic document used to prove the valid attribution of a public key to the identity of its holder. The certificate includes the public key and information about it, information about the identity of its owner (called the subject), and the digital signature of an entity that has verified the certificate's contents (called the issuer).
If the party examining the certificate trusts the issuer and finds the signature to be a valid signature of that issuer, then it can use the included public key to interact securely with the certificate's subject. In email encryption, code signing, and e-signature systems, a certificate's subject is typically a person or organization. However, in Transport Layer Security (TLS) a certificate's subject is typically a computer or other device, though TLS certificates may identify organizations or individuals in addition to their core role in identifying devices. TLS, sometimes called by its older name Secure Sockets Layer (SSL), is notable for being a part of HTTPS, a protocol for securely browsing the web.
In a typical public-key infrastructure (PKI) scheme, the certificate issuer is a certificate authority (CA), usually a company that charges customers a fee to issue certificates for them. By contrast, in a web of trust scheme, individuals sign each other's keys directly, in a format that performs a similar function to a public-key certificate. 
A public-key certificate is typically requested from a PKI using a CSR, which needs to be transferred using a secure certificate enrollment protocol such as CMP, EST, or ACME. The parties involved in the enrollment process must verify the authenticity, integrity, and authorization of the CSR, for which the certificate issuer bears the main responsibility.
In case of key compromise or other situations that may lead to unauthorized use, a certificate may need to be revoked.
The general format for public-key and attribute certificates is defined by X.509. For public-key certificates,  the format has been profiled by the IETF for Internet-related use cases, such as Public-Key Infrastructure (X.509).
