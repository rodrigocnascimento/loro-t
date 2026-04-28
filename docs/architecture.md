# Loro Event Transport Architecture

## Event Envelope

Todos os eventos enviados por `loro.nvim` e pelo script do WezTerm devem conter:

```json
{
  "event_id": "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
  "source": "nvim|wezterm",
  "event_type": "...",
  "payload": {}
}
```

- `event_id` é UUID v4 e é obrigatório.
- `event_id` é criado no cliente antes de qualquer tentativa de envio.

## Retry & Backoff (socket indisponível)

Clientes (`loro.nvim` e WezTerm) usam retry curto com backoff exponencial:

- Tentativas: 4
- Backoff inicial: 25ms
- Backoff máximo: 150ms
- Erros cobertos: socket indisponível, timeout de ACK, falha de conexão transitória

Se todas as tentativas falharem, o evento entra no fluxo offline (quando habilitado).

## ACK Protocol

Cada mensagem processada pelo daemon deve retornar ACK em JSON:

```json
{
  "ok": true,
  "reason": "accepted|duplicate_ignored|...",
  "event_id": "..."
}
```

ou

```json
{
  "ok": false,
  "reason": "invalid_json|missing_event_id|handler_rejected|...",
  "event_id": "..."
}
```

### Regras

- `ok=true`: evento foi aceito **ou** ignorado por deduplicação.
- `ok=false`: evento não foi processado com sucesso; cliente pode aplicar retry/offline policy.
- `reason` sempre obrigatório para facilitar diagnóstico.

## Deduplicação no daemon

O daemon mantém cache curto em memória com `event_id` recentes (FIFO):

- tamanho default: 500 IDs
- ao exceder limite, remove o mais antigo
- evento duplicado recebe `ok=true` + `reason=duplicate_ignored`

Isso garante idempotência prática durante retries/reconexões.

## Offline behavior

Clientes podem habilitar buffer local para eventos recentes:

- `enable_offline_buffer=true|false`
- `max_buffer_size` (default 100)

### Política

- Se envio falhar após retries e buffer estiver habilitado: evento é enfileirado.
- Se buffer estiver desabilitado: evento é descartado com log explícito.
- Se buffer exceder limite: descarta o evento mais antigo com log explícito contendo `dropped_event_id`.

## Failure Modes

### 1) Daemon restart

**Sintoma:** conexão quebra, eventos em voo podem falhar com timeout/EOF.  
**Comportamento esperado:** clientes fazem retry curto; se ainda falhar, entram no buffer offline (se habilitado). Após daemon voltar, `flush_buffer` reenvia.

### 2) Socket permission denied

**Sintoma:** erro imediato ao conectar (`EACCES`).  
**Comportamento esperado:** retries geralmente não resolvem; cliente registra erro, aplica política offline/descarta conforme configuração, sem loop infinito.

### 3) Integração temporariamente indisponível

**Sintoma:** produtor ou consumidor parcial fora do ar (ex.: WezTerm script carregado, daemon indisponível por manutenção).  
**Comportamento esperado:** retries + buffer local curto absorvem indisponibilidade breve. Em indisponibilidade prolongada, buffer roda FIFO e descarta explicitamente itens antigos com log.
