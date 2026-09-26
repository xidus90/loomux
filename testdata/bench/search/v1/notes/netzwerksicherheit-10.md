---
titel: Man-in-the-middle attack
quelle: https://en.wikipedia.org/wiki/Man-in-the-middle_attack
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerksicherheit
---

In cryptography and computer security, a man-in-the-middle (MITM) attack, or on-path attack, is a cyberattack where the attacker secretly relays and possibly alters the communications between two parties who believe that they are directly communicating with each other, where in actuality the attacker has inserted themselves between the two user parties.
One example of a MITM attack is active eavesdropping, in which the attacker makes independent connections with the victims and relays messages between them to make them believe they are talking directly to each other over a private connection, when in fact the entire conversation is controlled by the attacker. In this scenario, the attacker must be able to intercept all relevant messages passing between the two victims and inject new ones. This is straightforward in many circumstances; for example, an attacker within range of a Wi-Fi access point hosting a network without encryption could insert themselves as a man in the middle.
As it aims to circumvent mutual authentication, a MITM attack can succeed only when the attacker impersonates each endpoint sufficiently well to satisfy their expectations. Most cryptographic protocols include some form of endpoint authentication specifically to prevent MITM attacks. For example, TLS can authenticate one or both parties using a mutually trusted certificate authority.
