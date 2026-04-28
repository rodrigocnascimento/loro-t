# Health checks e critérios de transição

Este diretório define o contrato de saúde entre integrações (`docker`, `ollama`, etc.) e o daemon.

## Estrutura do resultado

Cada componente publica um `HealthCheckResult` com os campos:

- `component`: nome lógico da integração (`docker`, `ollama`, etc.).
- `status`: `ok`, `degraded`, `down` ou `unknown`.
- `last_success_at`: timestamp da última execução bem-sucedida.
- `last_error`: mensagem do último erro observado.
- `latency_ms`: latência do check mais recente em milissegundos.
- `ttl_seconds`: janela de validade esperada para dados de saúde.

## Máquina de estados de saúde

Critérios de transição implementados no scheduler:

1. **`unknown -> ok`**: primeiro check bem-sucedido.
2. **`ok -> degraded`**: primeira falha após período saudável.
3. **`degraded -> down`**: nova falha consecutiva (ou 3 falhas consecutivas no total).
4. **`down -> ok`**: qualquer sucesso subsequente.
5. **`* -> down` por staleness**: se `now - last_success_at > 2 * ttl_seconds`, estado é forçado para `down`.

## Backoff básico

Após falha, o daemon aumenta o intervalo de execução para `interval * 2^n` (limitado por `max_backoff`), onde `n` é o número de falhas consecutivas truncado em 3.
Após sucesso, o intervalo volta ao valor base.
