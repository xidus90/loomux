---
titel: Secure Shell
quelle: https://en.wikipedia.org/wiki/Secure_Shell
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerksicherheit
---

The Secure Shell Protocol (SSH Protocol) is a cryptographic network protocol for operating network services securely over an unsecured network. Its most notable applications are remote login and command-line execution.
SSH was designed for Unix-like operating systems as a replacement for Telnet and unsecured remote Unix shell protocols, such as the Berkeley Remote Shell (rsh) and the related rlogin and rexec protocols, which all use insecure, plaintext methods of authentication, such as passwords.
Since mechanisms like Telnet and Remote Shell are designed to access and operate remote computers, sending the authentication tokens (e.g. username and password) for this access to these computers across a public network in an unsecured way poses a great risk of third parties obtaining the password and achieving the same level of access to the remote system as the telnet user. Secure Shell mitigates this risk through the use of encryption mechanisms that are intended to hide the contents of the transmission from an observer, even if the observer has access to the entire data stream.
Finnish computer scientist Tatu Ylönen designed SSH in 1995 and provided an implementation in the form of two commands, ssh and slogin, as secure replacements for rsh and rlogin, respectively. Subsequent development of the protocol suite proceeded in several developer groups, producing several variants of implementation. The protocol specification distinguishes two major versions, referred to as SSH-1 and SSH-2. The most commonly implemented software stack is OpenSSH, released in 1999 as open-source software by the OpenBSD developers. Implementations are distributed for all types of operating systems in common use, including embedded systems.
SSH applications are based on a client–server architecture, connecting an SSH client instance with an SSH server. SSH operates as a layered protocol suite comprising three principal hierarchical components: the transport layer provides server authentication, confidentiality, and integrity; the user authentication protocol validates the user to the server; and the connection protocol multiplexes the encrypted tunnel into multiple logical communication channels.
