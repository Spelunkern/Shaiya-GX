# Discrepancias

Diferencias detectadas entre el cliente original, Rapture y este cliente.
No se corrige Rapture desde este proyecto.

| # | Tema | Original | Rapture | Este cliente |
|---|---|---|---|---|
| 1 | Secreto RSA de 0xA101 (`0x401160`) | 128 bytes aleatorios, solo se limpia el bit alto del último | `DecryptLoginRSA` reconstruye el secreto con `big.Int` y pierde los bytes altos en cero; la clave HMAC cambia y el login falla (≈1 de cada 128 logins con el cliente original: el byte alto queda en 0..127) | Igual al original. **Resuelto en Rapture** (commit `Keep the decrypted login secret 128 bytes wide`) |
